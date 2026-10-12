package relay

import (
	"bytes"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"willnorris.com/go/microformats"
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

func TestLandingDocumentIsMF2Entry(t *testing.T) {
	body, e := landingDocument(target)
	if e != nil {
		t.Fatal(e)
	}
	base, e := url.Parse(target)
	if e != nil {
		t.Fatal(e)
	}
	doc, e := html.Parse(bytes.NewReader(body))
	if e != nil {
		t.Fatal(e)
	}
	data := microformats.ParseNode(doc, base)
	if len(data.Items) != 1 {
		t.Fatalf("MF2 top-level entries = %d, want 1", len(data.Items))
	}
	entry := data.Items[0]
	if len(entry.Type) != 1 || entry.Type[0] != "h-entry" {
		t.Fatalf("entry type = %v, want [h-entry]", entry.Type)
	}
	if got := entry.Properties["name"]; len(got) != 1 || got[0] != landingTitle {
		t.Fatalf("entry name = %v, want [%q]", got, landingTitle)
	}
	content := entry.Properties["content"]
	if len(content) != 1 {
		t.Fatalf("entry content = %v, want one e-content", content)
	}
	value, ok := content[0].(map[string]string)
	if !ok {
		t.Fatalf("entry content type = %T, want map[string]string", content[0])
	}
	if !strings.Contains(value["html"], "Webmention") || !strings.Contains(value["html"], "https://indieweb.org/Webmention") {
		t.Fatalf("entry content omits description or its link: %s", value["html"])
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
		`<h1 class="p-name">indie-mezzanine</h1>`,
		`<article class="h-entry">`,
		`<div class="e-content">`,
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
