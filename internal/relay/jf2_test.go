package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func jf2Parse(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var f map[string]any
	if e := json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	return f
}

func jf2ChildrenOf(t *testing.T, f map[string]any) []any {
	t.Helper()
	c, _ := f["children"].([]any)
	if len(c) == 0 {
		t.Fatal("children欠落")
	}
	return c
}

// jf2ForbiddenKeysは保存済みmf2の内部表現と連絡先がjf2へ漏れないことを検査する。
func jf2ForbiddenKeys(t *testing.T, v any, path string) {
	t.Helper()
	switch x := v.(type) {
	case map[string]any:
		for _, k := range []string{"value", "id", "lang", "html", "email"} {
			if _, ok := x[k]; ok {
				t.Fatalf("%s に禁止キー %q", path, k)
			}
		}
		for k, c := range x {
			jf2ForbiddenKeys(t, c, path+"."+k)
		}
	case []any:
		for i, c := range x {
			jf2ForbiddenKeys(t, c, fmt.Sprintf("%s[%d]", path, i))
		}
	}
}

// jf2RichPostは著者card、複数写真、category、children、identity、連絡先、メディアを持つ固定fixture。
func jf2RichPost(t *testing.T) Post {
	t.Helper()
	body := `<html><head><link rel="collection" href="urn:uuid:` + id1 + `"></head><body><a href="` + target + `">relay</a><article class="h-entry">` +
		`<h1 class="p-name">題 &amp; 名</h1>` +
		`<div class="e-content">本文 &lt;script&gt;alert(1)&lt;/script&gt; &amp; 続き</div>` +
		`<a class="u-uid" href="/canonical">通し</a><a class="u-url" href="/canonical">URL</a>` +
		`<img class="u-photo" src="/photo1.jpg" alt="写真1"><img class="u-photo" src="/photo2.jpg" alt="写真2">` +
		`<span class="p-category">分類A</span><a class="p-category" href="/tag">分類B</a>` +
		`<p class="p-summary">要約 &amp; 詳細</p>` +
		`<a class="u-in-reply-to" href="https://origin.example/reply">返信</a>` +
		`<span class="p-author h-card"><span class="p-name">著者</span><a class="u-url" href="/author">author</a><img class="u-photo" src="/author.jpg" alt="著者写真"><a class="u-email" href="mailto:author@example.com">mail</a></span>` +
		`<time class="dt-published" datetime="2026-01-02T03:04:05Z">1月2日</time>` +
		`<video class="u-video" src="/clip.mp4"></video><audio class="u-audio" src="/sound.mp3"></audio>` +
		`<div class="h-cite"><span class="p-name">子項目</span><div class="e-content"><b>子内容</b></div></div>` +
		`</article></body></html>`
	r, e := Parse(page(body), target)
	if e != nil {
		t.Fatal(e)
	}
	return Post{Source: "https://source.example/post", MF2: string(r.MF2), Updated: time.Date(2026, 1, 3, 4, 5, 6, 0, time.UTC)}
}

func TestJF2RichProjection(t *testing.T) {
	p := jf2RichPost(t)
	b, e := JF2(target, id1, []Post{p}, 100)
	if e != nil {
		t.Fatal(e)
	}
	f := jf2Parse(t, b)
	if f["type"] != "feed" || f["name"] != "urn:uuid:"+id1 || f["url"] != target+"collections/"+id1+".jf2" {
		t.Fatalf("feed %#v", f)
	}
	if !reflect.DeepEqual(f["author"], map[string]any{"type": "card", "name": "indie-mezzanine"}) {
		t.Fatalf("feed author %#v", f["author"])
	}
	children := jf2ChildrenOf(t, f)
	if len(children) != 1 {
		t.Fatalf("children %#v", children)
	}
	entry, _ := children[0].(map[string]any)
	want := map[string]any{
		"audio":       map[string]any{"url": "https://source.example/sound.mp3"},
		"author":      map[string]any{"type": "card", "name": "著者", "url": "https://source.example/author", "photo": "https://source.example/author.jpg"},
		"category":    []any{"分類A", "分類B"},
		"children":    []any{map[string]any{"type": "cite", "name": "子項目", "content": map[string]any{"text": "子内容"}}},
		"content":     map[string]any{"text": "本文 <script>alert(1)</script> & 続き"},
		"in-reply-to": "https://origin.example/reply",
		"name":        "題 & 名",
		"photo":       "https://source.example/photo1.jpg",
		"published":   "2026-01-02T03:04:05Z",
		"summary":     "要約 & 詳細",
		"type":        "entry",
		"uid":         p.Source,
		"updated":     "2026-01-03T04:05:06Z",
		"url":         p.Source,
		"video":       map[string]any{"url": "https://source.example/clip.mp4"},
	}
	if !reflect.DeepEqual(entry, want) {
		t.Fatalf("entry\n got %#v\nwant %#v", entry, want)
	}
	jf2ForbiddenKeys(t, f, "$")
}

func TestJF2FallbacksAndEdges(t *testing.T) {
	fixed := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	p := Post{Source: "https://source.example/min", MF2: `{"type":["h-entry"],"properties":{}}`, Updated: fixed}
	b, e := JF2(target, id1, []Post{p}, 100)
	if e != nil {
		t.Fatal(e)
	}
	f := jf2Parse(t, b)
	entry := jf2ChildrenOf(t, f)[0].(map[string]any)
	want := map[string]any{
		"type":    "entry",
		"uid":     p.Source,
		"url":     p.Source,
		"name":    p.Source,
		"author":  map[string]any{"type": "card", "name": jf2AuthorFallback},
		"updated": "2026-02-03T04:05:06Z",
	}
	if !reflect.DeepEqual(entry, want) {
		t.Fatalf("欠落時の代替\n got %#v\nwant %#v", entry, want)
	}
	// 無効なpublishedは出さず、updatedはオフセットをUTCへ揃える。
	edge := Post{Source: "https://source.example/edge", MF2: `{"properties":{"published":["yesterday"],"updated":["2026-02-01T01:00:00+09:00"],"content":[{"value":""}],"photo":["https://source.example/only.jpg"]}}`, Updated: fixed}
	b, _ = JF2(target, id1, []Post{edge}, 100)
	entry = jf2ChildrenOf(t, jf2Parse(t, b))[0].(map[string]any)
	if _, ok := entry["published"]; ok {
		t.Fatalf("無効日時 %#v", entry)
	}
	if entry["updated"] != "2026-01-31T16:00:00Z" || !reflect.DeepEqual(entry["content"], map[string]any{"text": ""}) || entry["photo"] != "https://source.example/only.jpg" {
		t.Fatalf("日時と本文 %#v", entry)
	}
	// 値が無いcontentは生成しない。
	empty := Post{Source: "https://source.example/empty", MF2: `{"properties":{"content":[]}}`, Updated: fixed}
	b, _ = JF2(target, id1, []Post{empty}, 100)
	entry = jf2ChildrenOf(t, jf2Parse(t, b))[0].(map[string]any)
	if _, ok := entry["content"]; ok {
		t.Fatalf("空content %#v", entry)
	}
}

func TestJF2TextOnlyAndDeterminism(t *testing.T) {
	fixed := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	p := Post{Source: "https://source.example/text", MF2: `{"properties":{"content":[{"value":"<script>alert(1)</script> & tail"}]}}`, Updated: fixed}
	newer := Post{Source: "https://source.example/newer", MF2: `{"properties":{"content":[{"value":"新"}]}}`, Updated: fixed.Add(time.Hour)}
	b, e := JF2(target, id1, []Post{p, newer}, 100)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(b, []byte("<script>")) {
		t.Fatal("HTML表現を出力")
	}
	var entry map[string]any
	for _, c := range jf2ChildrenOf(t, jf2Parse(t, b)) {
		if m := c.(map[string]any); m["uid"] == p.Source {
			entry = m
		}
	}
	if entry == nil {
		t.Fatal("entry欠落")
	}
	content, ok := entry["content"].(map[string]any)
	if !ok || content["text"] != "<script>alert(1)</script> & tail" {
		t.Fatalf("本文 %#v", entry)
	}
	if _, ok := content["html"]; ok {
		t.Fatalf("htmlキー %#v", content)
	}
	b1, _ := JF2(target, id1, []Post{p, newer}, 100)
	b2, _ := JF2(target, id1, []Post{newer, p}, 100)
	if !bytes.Equal(b1, b2) {
		t.Fatal("再取得または入力順で不安定出力")
	}
}

func TestJF2OrderingLimitAndTimes(t *testing.T) {
	s := store(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range []string{"c", "b", "a"} {
		r := mustRecord(t, id1, name)
		if e := s.Apply(ctx, Job{Source: "https://source.example/" + name, Target: target}, "https://source.example/"+name, &r, now); e != nil {
			t.Fatal(e)
		}
	}
	_, ps, e := s.Feed(ctx, id1, 2)
	if e != nil {
		t.Fatal(e)
	}
	b, e := JF2(target, id1, ps, 1)
	if e != nil {
		t.Fatal(e)
	}
	children := jf2ChildrenOf(t, jf2Parse(t, b))
	if len(children) != 1 || !strings.HasSuffix(children[0].(map[string]any)["uid"].(string), "/a") {
		t.Fatalf("順序/上限 %#v", children)
	}
	b2, _ := JF2(target, id1, append([]Post(nil), ps...), 1)
	if !bytes.Equal(b, b2) {
		t.Fatal("再取得の不安定出力")
	}
	p := Post{Source: "https://source.example/new", Updated: now, MF2: `{"properties":{"updated":["2026-02-01T01:00:00+09:00"],"published":["2025-12-31T12:00:00Z"]}}`}
	b, _ = JF2(target, id1, []Post{p}, 100)
	entry := jf2ChildrenOf(t, jf2Parse(t, b))[0].(map[string]any)
	if entry["updated"] != "2026-01-31T16:00:00Z" || entry["published"] != "2025-12-31T12:00:00Z" {
		t.Fatalf("Atomと同じ日時文字列でない %#v", entry)
	}
}

func TestJF2HTTPContract(t *testing.T) {
	s := store(t)
	a, e := NewApp(config(t), s, resultFetcher{})
	if e != nil {
		t.Fatal(e)
	}
	r := mustRecord(t, id1, "x")
	if e := s.Apply(context.Background(), Job{Source: "https://source.example/post", Target: target}, "https://source.example/post", &r, time.Now()); e != nil {
		t.Fatal(e)
	}
	h := a.Handler()
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "http://relay.example"+path, nil))
		return w
	}
	w := get("/collections/" + id1 + ".jf2")
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/jf2feed+json" {
		t.Fatalf("jf2 %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	f := jf2Parse(t, w.Body.Bytes())
	if f["name"] != "urn:uuid:"+id1 || len(jf2ChildrenOf(t, f)) != 1 {
		t.Fatalf("%#v", f)
	}
	if w := get("/collections/" + id1 + ".atom"); w.Code != 200 || !strings.Contains(w.Header().Get("Content-Type"), "application/atom+xml") {
		t.Fatalf("atom %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	for _, path := range []string{"/collections/" + id2 + ".jf2", "/collections/nope.jf2", "/collections/" + id1 + ".json", "/collections/" + id1} {
		if w := get(path); w.Code != 404 {
			t.Fatalf("%s %d", path, w.Code)
		}
	}
}

func TestJF2FormatParity(t *testing.T) {
	s := store(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	record := func(body string) Record {
		t.Helper()
		r, e := Parse(page(body), target)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	posts := []struct {
		source string
		body   string
		at     time.Time
	}{
		{"https://source.example/a", fixture(id1, "a"), now},
		{"https://source.example/b", strings.Replace(fixture(id1, "b"), "</article>", `<time class="dt-published" datetime="2025-12-31T21:00:00+09:00">日時</time></article>`, 1), now.Add(time.Hour)},
		{"https://source.example/c", fixture(id1, "c"), now.Add(2 * time.Hour)},
	}
	for _, p := range posts {
		r := record(p.body)
		if e := s.Apply(ctx, Job{Source: p.source, Target: target}, p.source, &r, p.at); e != nil {
			t.Fatal(e)
		}
	}
	up, ps, e := s.Feed(ctx, id1, 100)
	if e != nil {
		t.Fatal(e)
	}
	atomBefore, e := Atom(target, id1, up, append([]Post(nil), ps...), 100)
	if e != nil {
		t.Fatal(e)
	}
	jf2Body, e := JF2(target, id1, append([]Post(nil), ps...), 100)
	if e != nil {
		t.Fatal(e)
	}
	atomAfter, e := Atom(target, id1, up, append([]Post(nil), ps...), 100)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(atomBefore, atomAfter) {
		t.Fatal("JF2追加でAtom変動")
	}
	var af struct {
		Entries []struct {
			ID        string `xml:"id"`
			Published string `xml:"published"`
			Updated   string `xml:"updated"`
		} `xml:"entry"`
	}
	if e = xml.Unmarshal(atomBefore, &af); e != nil {
		t.Fatal(e)
	}
	f := jf2Parse(t, jf2Body)
	children := jf2ChildrenOf(t, f)
	if len(af.Entries) != len(children) {
		t.Fatalf("集合 %d %d", len(af.Entries), len(children))
	}
	for i, ae := range af.Entries {
		ce := children[i].(map[string]any)
		uid, _ := ce["uid"].(string)
		updated, _ := ce["updated"].(string)
		published, _ := ce["published"].(string)
		if ae.ID != uid || ae.Updated != updated || ae.Published != published {
			t.Fatalf("形式間不一致 %d: atom=%#v jf2=%#v", i, ae, ce)
		}
	}
	other, e := JF2("https://other.example/", id1, append([]Post(nil), ps...), 100)
	if e != nil {
		t.Fatal(e)
	}
	of := jf2Parse(t, other)
	if of["name"] != f["name"] || of["url"] == f["url"] {
		t.Fatalf("別リレー %#v", of)
	}
}
