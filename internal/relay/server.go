package relay

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type App struct {
	Config    Config
	Store     *Store
	Fetcher   Fetcher
	active    atomic.Int32
	unhealthy atomic.Bool
	wg        sync.WaitGroup
}

func NewApp(c Config, s *Store, f Fetcher) (*App, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	if s == nil || f == nil {
		return nil, fmt.Errorf("処理依存が必要")
	}
	return &App{Config: c, Store: s, Fetcher: f}, nil
}
func (a *App) Process(ctx context.Context, j Job) error {
	p, e := a.Fetcher.Fetch(ctx, j.Source)
	if e != nil {
		return e
	}
	if p.Status == 404 || p.Status == 410 {
		return a.Store.Apply(ctx, j, p.URL, nil, time.Now())
	}
	if p.Status != 200 {
		return fmt.Errorf("原本HTTP %d", p.Status)
	}
	r, e := Parse(p, j.Target)
	if errors.Is(e, ErrIneligible) {
		return a.Store.Apply(ctx, j, p.URL, nil, time.Now())
	}
	if e != nil {
		return e
	}
	return a.Store.Apply(ctx, j, p.URL, &r, time.Now())
}
func (a *App) Start(ctx context.Context) error {
	if e := a.Store.Recover(ctx, a.Config.Attempts); e != nil {
		return e
	}
	for i := 0; i < a.Config.Workers; i++ {
		a.wg.Add(1)
		a.active.Add(1)
		go func() {
			defer a.wg.Done()
			defer a.active.Add(-1)
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					j, e := a.Store.Claim(ctx, time.Now())
					if e == sql.ErrNoRows {
						continue
					}
					if e != nil {
						if ctx.Err() == nil {
							a.unhealthy.Store(true)
						}
						continue
					}
					e = a.Process(ctx, j)
					if ctx.Err() != nil {
						return
					}
					if e = a.Store.Finish(ctx, j, e, a.Config.Attempts, time.Now()); e != nil {
						a.unhealthy.Store(true)
						log.Print("通知処理状態の保存失敗")
					}
				}
			}
		}()
	}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if e := a.Store.Cleanup(ctx, time.Now(), a.Config.Retention); e != nil {
					a.unhealthy.Store(true)
				}
			}
		}
	}()
	return nil
}
func (a *App) Wait() { a.wg.Wait() }
func (a *App) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		endpoint := strings.TrimRight(a.Config.PublicURL, "/") + "/webmention"
		w.Header().Set("Link", "<"+endpoint+">; rel=\"webmention\"")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><html lang="ja"><head><meta charset="utf-8"><title>mezzanine</title><link rel="webmention" href="%s"></head><body><h1>mezzanine</h1><p>公開ページの単一h-entryとUUID collectionをAtomに束ねるリレーです。Webmentionの受付は掲載の保証ではありません。</p></body></html>`, html.EscapeString(endpoint))
	})
	m.HandleFunc("POST /webmention", func(w http.ResponseWriter, r *http.Request) {
		media, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || media != "application/x-www-form-urlencoded" {
			http.Error(w, "フォーム形式が必要", 400)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, a.Config.MaxForm)
		if e = r.ParseForm(); e != nil {
			http.Error(w, "フォームが不正または上限超過", 400)
			return
		}
		if len(r.PostForm["source"]) != 1 || len(r.PostForm["target"]) != 1 {
			http.Error(w, "source/targetは一つずつ必要", 400)
			return
		}
		source, target := r.PostForm.Get("source"), r.PostForm.Get("target")
		if _, e = sourceURL(source); e != nil || source == target || target != a.Config.PublicURL {
			http.Error(w, "source/targetが不正", 400)
			return
		}
		// targetだけとfragmentが異なるsourceも同じ資源として拒否する。
		u, _ := url.Parse(source)
		u.Fragment = ""
		if u.String() == target {
			http.Error(w, "sourceとtargetが同じ", 400)
			return
		}
		e = a.Store.Enqueue(r.Context(), source, target, a.Config.Queue, time.Now())
		if errors.Is(e, ErrFull) {
			http.Error(w, "通知キュー満杯", 503)
			return
		}
		if e != nil {
			http.Error(w, "永続受付失敗", 503)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(202)
		fmt.Fprintln(w, "通知を永続受付しました。掲載は原本検証後です。")
	})
	m.HandleFunc("GET /collections/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !strings.HasSuffix(name, ".atom") {
			http.NotFound(w, r)
			return
		}
		raw := strings.TrimSuffix(name, ".atom")
		id, e := Collection("urn:uuid:" + raw)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		updated, posts, e := a.Store.Feed(r.Context(), id, a.Config.FeedLimit)
		if errors.Is(e, ErrUnknown) {
			http.NotFound(w, r)
			return
		}
		if e != nil {
			http.Error(w, "保存データの取得失敗", 503)
			return
		}
		b, e := Atom(a.Config.PublicURL, id, updated, posts, a.Config.FeedLimit)
		if e != nil {
			http.Error(w, "Atom生成失敗", 500)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		w.Write(b)
	})
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if a.active.Load() != int32(a.Config.Workers) || a.unhealthy.Load() || a.Store.Health(ctx) != nil {
			http.Error(w, "処理系または保存先が利用不能", 503)
			return
		}
		fmt.Fprintln(w, "ok")
	})
	return m
}
func (a *App) Server() *http.Server {
	return &http.Server{Addr: a.Config.Listen, Handler: a.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
}
