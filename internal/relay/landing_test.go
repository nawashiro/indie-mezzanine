package relay

import (
	"bytes"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestLandingDocumentDeterministic(t *testing.T) {
	first, e := landingDocument(target)
	if e != nil {
		t.Fatal(e)
	}
	if len(first) == 0 || !bytes.Contains(first, []byte("<h1")) {
		t.Fatalf("描画結果が空か見出しなし: %q", first)
	}
	second, e := landingDocument(target)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("同一入力で出力が変動")
	}
}

func TestLandingResolvesDestinations(t *testing.T) {
	base, e := url.Parse(target)
	if e != nil {
		t.Fatal(e)
	}
	source := []byte("[受付](/webmention)\n\n[購読](/collections/" + id1 + ".atom)\n\n[外部](https://nawashiro.dev/x)\n\n[メール](mailto:a@example.com)\n\n[断片](#section)\n\n![画像](/image.png)\n")
	body, e := renderMarkdown(source, base)
	if e != nil {
		t.Fatal(e)
	}
	got := string(body)
	for _, want := range []string{
		`href="https://relay.example/webmention"`,
		`href="https://relay.example/collections/` + id1 + `.atom"`,
		`href="https://nawashiro.dev/x"`,
		`href="mailto:a@example.com"`,
		`href="#section"`,
		`src="https://relay.example/image.png"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("宛先解決の欠落 %q in %s", want, got)
		}
	}
}

func TestLandingDocumentRejectsBadBase(t *testing.T) {
	if _, e := landingDocument("not-a-url"); e == nil {
		t.Fatal("不正な公開URLを基底として受理")
	}
}

func TestRootPageServesLanding(t *testing.T) {
	a, e := NewApp(config(t), store(t), resultFetcher{})
	if e != nil {
		t.Fatal(e)
	}
	if len(a.landing) == 0 {
		t.Fatal("起動時に描画されていない")
	}
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest("GET", "http://spoof.example/", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatal(ct)
	}
	body := w.Body.String()
	for _, want := range []string{
		"<title>indie-mezzanine</title>",
		"<h1>indie-mezzanine</h1>",
		"投稿",
		"閲覧",
		`rel="webmention"`,
		`href="` + target + `webmention"`,
		`href="` + target + `collections/`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("本文の欠落 %q in %s", want, body)
		}
	}
	if !bytes.Equal(w.Body.Bytes(), a.landing) {
		t.Fatal("キャッシュではなく再描画した")
	}
}
