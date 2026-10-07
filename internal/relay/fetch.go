package relay

import (
	"context"
	"fmt"
	"github.com/doyensec/safeurl"
	"io"
	"mime"
	"net/http"
	"net/url"
	"time"
)

type Page struct {
	URL    string
	Status int
	Body   []byte
}
type Fetcher interface {
	Fetch(context.Context, string) (Page, error)
}
type SafeFetcher struct {
	client *safeurl.WrappedClient
	limit  int64
}

func sourceURL(raw string) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return nil, fmt.Errorf("取得URLのscheme/host/認証情報が不正")
	}
	if p := u.Port(); p != "" && p != "80" && p != "443" {
		return nil, fmt.Errorf("許可外ポート")
	}
	return u, nil
}
func redirectCheck(max int) func(*http.Request, []*http.Request) error {
	return func(r *http.Request, via []*http.Request) error {
		if len(via) > max {
			return fmt.Errorf("転送上限")
		}
		_, e := sourceURL(r.URL.String())
		return e
	}
}
func NewSafeFetcher(c Config) (*SafeFetcher, error) {
	if c.FetchTimeout <= 0 || c.MaxBody <= 0 || c.Redirects <= 0 {
		return nil, fmt.Errorf("取得上限が不正")
	}
	tr := &http.Transport{Proxy: nil, ResponseHeaderTimeout: c.FetchTimeout, TLSHandshakeTimeout: c.FetchTimeout, IdleConnTimeout: 30 * time.Second, MaxIdleConns: 4, MaxIdleConnsPerHost: 2, MaxResponseHeaderBytes: 32 << 10}
	cfg := safeurl.GetConfigBuilder().SetAllowedSchemes("http", "https").SetAllowedPorts(80, 443).EnableIPv6(true).SetTimeout(c.FetchTimeout).SetCheckRedirect(redirectCheck(c.Redirects)).SetTransport(tr).Build()
	return &SafeFetcher{safeurl.Client(cfg), c.MaxBody}, nil
}
func (f *SafeFetcher) Fetch(ctx context.Context, raw string) (Page, error) {
	if _, e := sourceURL(raw); e != nil {
		return Page{}, e
	}
	req, e := http.NewRequestWithContext(ctx, "GET", raw, nil)
	if e != nil {
		return Page{}, e
	}
	req.Header.Set("Accept", "text/html, application/xhtml+xml;q=0.9")
	req.Header.Set("User-Agent", "mezzanine Webmention receiver")
	res, e := f.client.Do(req)
	if e != nil {
		return Page{}, e
	}
	defer res.Body.Close()
	p := Page{URL: res.Request.URL.String(), Status: res.StatusCode}
	if res.StatusCode != 200 {
		return p, nil
	}
	media, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil || (media != "text/html" && media != "application/xhtml+xml") {
		return Page{}, fmt.Errorf("原本のContent-TypeはHTMLが必要")
	}
	body, e := io.ReadAll(io.LimitReader(res.Body, f.limit+1))
	if e != nil {
		return Page{}, e
	}
	if int64(len(body)) > f.limit {
		return Page{}, fmt.Errorf("展開後本文の上限超過")
	}
	p.Body = body
	return p, nil
}
