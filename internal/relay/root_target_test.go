package relay

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func rootNotify(a *App, source, target string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/webmention", strings.NewReader(url.Values{"source": {source}, "target": {target}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	return w
}

func TestRootAdmission(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		base := scheme + "://relay.example"
		for _, configured := range []string{base, base + "/"} {
			for _, submitted := range []string{base, base + "/"} {
				t.Run(configured+"->"+submitted, func(t *testing.T) {
					c, s := config(t), store(t)
					defer s.Close()
					c.PublicURL = configured
					f := &countFetcher{}
					a, e := NewApp(c, s, f)
					if e != nil {
						t.Fatal(e)
					}
					for _, source := range []string{base, base + "/", base + "#section", base + "/#section", base + "#", base + "/#"} {
						if w := rootNotify(a, source, submitted); w.Code != 400 {
							t.Fatal(source, w.Code)
						}
					}
					var n int
					if e := s.db.QueryRow("SELECT count(*) FROM jobs").Scan(&n); e != nil || n != 0 || f.calls != 0 {
						t.Fatal(n, e)
					}
					for _, source := range []string{"https://source.example/post#original", base + "/post", base + "/post/", base + "/?x"} {
						if w := rootNotify(a, source, submitted); w.Code != 202 {
							t.Fatal(source, w.Code, w.Body.String())
						}
						var gotSource, gotTarget string
						if e := s.db.QueryRow("SELECT source,target FROM jobs ORDER BY id DESC LIMIT 1").Scan(&gotSource, &gotTarget); e != nil || gotSource != source || gotTarget != base+"/" {
							t.Fatal(gotSource, gotTarget, e)
						}
					}
				})
			}
		}
	}
}

func TestRootTargetRejected(t *testing.T) {
	c, s := config(t), store(t)
	defer s.Close()
	f := &countFetcher{}
	a, _ := NewApp(c, s, f)
	for _, target := range []string{"https://other.example/", "http://relay.example/", "https://relay.example:443/", "https://RELAY.example/", "https://relay.example/post", "https://relay.example//", "https://relay.example/%2f", "https://relay.example/?", "https://relay.example/?x", "https://relay.example/#x", "https://user@relay.example/", ":bad"} {
		if w := rootNotify(a, "https://source.example/p", target); w.Code != 400 {
			t.Fatal(target, w.Code)
		}
	}
	var n int
	if e := s.db.QueryRow("SELECT count(*) FROM jobs").Scan(&n); e != nil || n != 0 || f.calls != 0 {
		t.Fatal(n, e)
	}
}

func TestRootLinks(t *testing.T) {
	for _, target := range []string{"https://relay.example", "https://relay.example/"} {
		for _, link := range []string{"https://relay.example", "https://relay.example/", "/", "../", "https://relay.example/#"} {
			body := strings.Replace(fixture(id1, "ok"), `href="https://relay.example/"`, `href="`+link+`"`, 1)
			p := page(body)
			p.URL = "https://relay.example/posts/p"
			if _, e := Parse(p, target); e != nil {
				t.Fatal(target, link, e)
			}
		}
		body := strings.Replace(fixture(id1, "ok"), "<head>", `<head><base href="https://relay.example/dir/">`, 1)
		body = strings.Replace(body, `href="https://relay.example/"`, `href="../"`, 1)
		if _, e := Parse(page(body), target); e != nil {
			t.Fatal(e)
		}
		for _, link := range []string{"https://other.example/", "http://relay.example/", "https://relay.example:443/", "https://relay.example/post", "https://relay.example/post/", "https://relay.example//", "https://relay.example/%2f", "https://relay.example/?x", "https://relay.example/?", "https://relay.example/#x"} {
			body := strings.Replace(fixture(id1, "ok"), `href="https://relay.example/"`, `href="`+link+`"`, 1)
			if _, e := Parse(page(body), target); !errors.Is(e, ErrIneligible) {
				t.Fatal(link, e)
			}
		}
	}
}

func TestRootJobsAcrossRestart(t *testing.T) {
	ctx := context.Background()
	for _, configured := range []string{"https://relay.example", "https://relay.example/"} {
		for _, oldTarget := range []string{"https://relay.example", "https://relay.example/"} {
			for _, state := range []string{"pending", "running"} {
				t.Run(configured+oldTarget+state, func(t *testing.T) {
					c := config(t)
					c.PublicURL = configured
					s, e := OpenStore(c.DataDir)
					if e != nil {
						t.Fatal(e)
					}
					source := "https://source.example/p"
					now := time.Now()
					if e := s.Enqueue(ctx, source, oldTarget, 10, now); e != nil {
						t.Fatal(e)
					}
					if state == "running" {
						if _, e := s.Claim(ctx, now); e != nil {
							t.Fatal(e)
						}
					}
					s.Close()
					s, e = OpenStore(c.DataDir)
					if e != nil {
						t.Fatal(e)
					}
					defer s.Close()
					if e := s.Recover(ctx, c.Attempts); e != nil {
						t.Fatal(e)
					}
					f := &countFetcher{p: page(strings.Replace(fixture(id1, "original"), `href="https://relay.example/"`, `href="https://relay.example"`, 1))}
					f.p.Status = 200
					a, _ := NewApp(c, s, f)
					j, e := s.Claim(ctx, time.Now())
					if e != nil || j.Target != oldTarget {
						t.Fatal(j, e)
					}
					if e := a.Process(ctx, j); e != nil {
						t.Fatal(e)
					}
					if e := s.Finish(ctx, j, nil, c.Attempts, time.Now()); e != nil {
						t.Fatal(e)
					}
					_, before, e := s.Feed(ctx, id1, 10)
					if e != nil || len(before) != 1 || before[0].Target != oldTarget {
						t.Fatal(before, e)
					}
					for _, submitted := range []string{"https://relay.example", "https://relay.example/"} {
						if w := rootNotify(a, source, submitted); w.Code != http.StatusOK {
							t.Fatal(w.Code)
						}
					}
					_, after, e := s.Feed(ctx, id1, 10)
					if e != nil || after[0] != before[0] || f.calls != 1 {
						t.Fatal(after, e, f.calls)
					}
					var version int
					if e := s.db.QueryRow("PRAGMA user_version").Scan(&version); e != nil || version != 2 {
						t.Fatal(version, e)
					}
				})
			}
		}
	}
}

func TestRootSourceIdentityAndFailedRenotify(t *testing.T) {
	ctx := context.Background()
	c, s := config(t), store(t)
	defer s.Close()
	c.PublicURL = "https://relay.example"
	f := &countFetcher{p: page(fixture(id1, "ok"))}
	f.p.Status = 404
	a, _ := NewApp(c, s, f)
	for _, source := range []string{"https://source.example", "https://source.example/", "https://source.example/post", "https://source.example/post/"} {
		if w := rootNotify(a, source, "https://relay.example"); w.Code != 202 {
			t.Fatal(w.Code)
		}
		j, e := s.Claim(ctx, time.Now())
		if e != nil {
			t.Fatal(e)
		}
		if source == "https://source.example" {
			if e := s.Finish(ctx, j, a.Process(ctx, j), c.Attempts, time.Now()); e != nil {
				t.Fatal(e)
			}
			if w := rootNotify(a, source, "https://relay.example/"); w.Code != 202 {
				t.Fatal(w.Code)
			}
			j, e = s.Claim(ctx, time.Now())
			if e != nil {
				t.Fatal(e)
			}
		}
		f.p.Status = 200
		if e := a.Process(ctx, j); e != nil {
			t.Fatal(e)
		}
		if e := s.Finish(ctx, j, nil, c.Attempts, time.Now()); e != nil {
			t.Fatal(e)
		}
	}
	_, posts, e := s.Feed(ctx, id1, 10)
	if e != nil || len(posts) != 4 {
		t.Fatal(posts, e)
	}
	if _, e := s.Claim(ctx, time.Now()); e != sql.ErrNoRows {
		t.Fatal(e)
	}
}
