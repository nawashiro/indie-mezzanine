package relay

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHTMLMediaType(t *testing.T) {
	for _, media := range []string{"text/html; charset=utf-8", "application/xhtml+xml", "application/json", "text/plain", ""} {
		t.Run(media, func(t *testing.T) {
			f, _ := NewSafeFetcher(config(t))
			f.client.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {media}}, Body: io.NopCloser(strings.NewReader(fixture(id1, "x"))), Request: r}, nil
			})
			p, e := f.Fetch(context.Background(), "https://source.example/post")
			allowed := strings.HasPrefix(media, "text/html") || media == "application/xhtml+xml"
			if allowed {
				if e != nil || len(p.Body) == 0 {
					t.Fatal(e)
				}
			} else if e == nil {
				t.Fatal("非HTMLを受理")
			}
		})
	}
}
func TestInvalidFetchLimits(t *testing.T) {
	for _, field := range []string{"timeout", "body", "redirects"} {
		c := config(t)
		switch field {
		case "timeout":
			c.FetchTimeout = 0
		case "body":
			c.MaxBody = 0
		case "redirects":
			c.Redirects = 0
		}
		if _, e := NewSafeFetcher(c); e == nil {
			t.Fatal(field)
		}
	}
}
