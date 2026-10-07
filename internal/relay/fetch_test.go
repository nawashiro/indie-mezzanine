package relay

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/doyensec/safeurl"
	"golang.org/x/net/dns/dnsmessage"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestSafeFetcherRejects(t *testing.T) {
	c := config(t)
	c.FetchTimeout = 300 * time.Millisecond
	f, e := NewSafeFetcher(c)
	if e != nil {
		t.Fatal(e)
	}
	defer f.client.CloseIdleConnections()
	for _, raw := range []string{"ftp://example.org/", "http://user:pass@example.org/", "http://@example.org/", "http://example.org:8080/", "http://127.0.0.1/", "http://10.0.0.1/", "http://172.16.0.1/", "http://192.168.0.1/", "http://169.254.169.254/", "http://100.64.0.1/", "http://[::1]/", "http://[::ffff:127.0.0.1]/", "http://[fc00::1]/", "http://[fe80::1]/", "http://localhost/"} {
		t.Run(raw, func(t *testing.T) {
			if _, e = f.Fetch(context.Background(), raw); e == nil {
				t.Fatal("安全拒否なし")
			}
		})
	}
	for _, raw := range []string{"http://example.org/", "https://example.org/", "http://example.org:80/", "https://example.org:443/"} {
		if _, e := sourceURL(raw); e != nil {
			t.Fatal(raw, e)
		}
	}
	tr := f.client.Client.Transport.(*http.Transport)
	if tr.Proxy != nil {
		t.Fatal("環境proxy有効")
	}
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	_, e = f.Fetch(context.Background(), "http://127.0.0.1/")
	var ipErr *safeurl.AllowedIPError
	if !errors.As(e, &ipErr) {
		t.Fatalf("接続時IP拒否ではない %v", e)
	}
}
func TestRedirectChecks(t *testing.T) {
	for _, location := range []string{"http://user:pass@example.org/", "ftp://example.org/", "http://example.org:8080/", "http://127.0.0.1/", "http://[::1]/"} {
		t.Run(location, func(t *testing.T) {
			c := config(t)
			c.FetchTimeout = 300 * time.Millisecond
			f, _ := NewSafeFetcher(c)
			safeTransport := f.client.Client.Transport
			var forbidden atomic.Int32
			f.client.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host == "start.example" {
					return &http.Response{StatusCode: 302, Header: http.Header{"Location": {location}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
				}
				res, e := safeTransport.RoundTrip(r)
				if e == nil {
					forbidden.Add(1)
				}
				return res, e
			})
			if _, e := f.Fetch(context.Background(), "http://start.example/"); e == nil {
				t.Fatal("redirect受理")
			}
			if forbidden.Load() != 0 {
				t.Fatal("拒否先到達")
			}
		})
	}
	c := config(t)
	c.Redirects = 2
	f, _ := NewSafeFetcher(c)
	var calls atomic.Int32
	f.client.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		n := calls.Add(1)
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": {fmt.Sprintf("http://start.example/%d", n)}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	if _, e := f.Fetch(context.Background(), "http://start.example/"); e == nil || calls.Load() != 3 {
		t.Fatal("回数上限", e, calls.Load())
	}
}
func TestBodyAndTimeLimits(t *testing.T) {
	for _, kind := range []string{"gzip", "chunked", "slow headers", "slow body"} {
		t.Run(kind, func(t *testing.T) {
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				switch kind {
				case "gzip":
					w.Header().Set("Content-Encoding", "gzip")
					var b bytes.Buffer
					g := gzip.NewWriter(&b)
					g.Write([]byte(strings.Repeat("x", 300)))
					g.Close()
					w.Write(b.Bytes())
				case "chunked":
					w.(http.Flusher).Flush()
					w.Write([]byte(strings.Repeat("x", 300)))
				case "slow headers":
					time.Sleep(100 * time.Millisecond)
					fmt.Fprint(w, "ok")
				case "slow body":
					w.(http.Flusher).Flush()
					time.Sleep(100 * time.Millisecond)
					fmt.Fprint(w, "ok")
				}
			}))
			defer source.Close()
			c := config(t)
			c.MaxBody = 100
			c.FetchTimeout = 30 * time.Millisecond
			f, _ := NewSafeFetcher(c)
			// 本文・時間経路のみ局所HTTPへ注入する。IP制約は別テストで本番transportを使う。
			tr := source.Client().Transport
			f.client.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				clone := r.Clone(r.Context())
				u := *r.URL
				u.Scheme = "http"
				u.Host = strings.TrimPrefix(source.URL, "http://")
				clone.URL = &u
				return tr.RoundTrip(clone)
			})
			if _, e := f.Fetch(context.Background(), "http://source.example/"); e == nil {
				t.Fatal("上限を超えて成功")
			}
		})
	}
}
func TestFetchDNSChangeIsolated(t *testing.T) {
	if os.Getenv("MEZZANINE_DNS_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestFetchDNSChangeIsolated$", "-test.timeout=10s")
		cmd.Env = append(os.Environ(), "MEZZANINE_DNS_CHILD=1")
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("隔離DNS検証: %v\n%s", e, out)
		}
		return
	}
	conn, e := net.ListenPacket("udp", ":8053")
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	var internal atomic.Bool
	var queries atomic.Int32
	done := make(chan struct{})
	defer close(done)
	go func() {
		buf := make([]byte, 2048)
		for {
			n, addr, e := conn.ReadFrom(buf)
			if e != nil {
				return
			}
			var p dnsmessage.Parser
			h, e := p.Start(buf[:n])
			if e != nil {
				continue
			}
			q, e := p.Question()
			if e != nil {
				continue
			}
			queries.Add(1)
			builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{ID: h.ID, Response: true, RecursionDesired: true, RecursionAvailable: true})
			builder.StartQuestions()
			builder.Question(q)
			builder.StartAnswers()
			if q.Type == dnsmessage.TypeA {
				ip := [4]byte{8, 8, 8, 8}
				if internal.Load() {
					ip = [4]byte{127, 0, 0, 1}
				}
				builder.AResource(dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 0}, dnsmessage.AResource{A: ip})
			}
			b, e := builder.Finish()
			if e == nil {
				conn.WriteTo(b, addr)
			}
		}
	}()
	// safeurlのテストresolverは固定8053なので別プロセスで直列実行。
	// Client.Getを直接使い、競合のあるテスト専用tracerを設置しない。
	// 接続時IP検査を含むtransportとresolverはsafeurlのものを維持する。
	cfg := safeurl.GetConfigBuilder().SetAllowedPorts(80, 443).SetAllowedSchemes("http", "https").SetTimeout(150 * time.Millisecond).EnableTestMode(true).SetCheckRedirect(redirectCheck(5)).Build()
	client := safeurl.Client(cfg)
	defer client.CloseIdleConnections()
	_, first := client.Client.Get("http://changing.test/")
	var blocked *safeurl.AllowedIPError
	if errors.As(first, &blocked) {
		t.Fatalf("公開IPを拒否 %v", first)
	}
	if queries.Load() == 0 {
		t.Fatal("DNS未実行")
	}
	client.CloseIdleConnections()
	internal.Store(true)
	_, second := client.Client.Get("http://changing.test/")
	if !errors.As(second, &blocked) {
		t.Fatalf("変更後の内部IPが接続時拒否されない %v", second)
	}
}
