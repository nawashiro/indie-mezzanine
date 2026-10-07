package relay

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const id1 = "550e8400-e29b-41d4-a716-446655440000"
const id2 = "550e8400-e29b-41d4-a716-446655440001"
const target = "https://relay.example/"

func fixture(id, body string) string {
	return `<html><head><link rel="collection" href="urn:uuid:` + id + `"></head><body><a href="` + target + `">relay</a><article class="h-entry"><h1 class="p-name">題 &amp; 名</h1><div class="e-content">` + body + `</div><a class="p-author h-card" href="https://author.example/"><span class="p-name">著者</span><span class="e-note"><b>紹介</b></span></a></article></body></html>`
}
func page(body string) Page {
	return Page{URL: "https://source.example/post", Status: 200, Body: []byte(body)}
}
func config(t *testing.T) Config {
	t.Helper()
	c := DefaultConfig()
	c.PublicURL = target
	c.DataDir = t.TempDir()
	return c
}
func store(t *testing.T) *Store {
	t.Helper()
	s, e := OpenStore(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func mustRecord(t *testing.T, id, body string) Record {
	t.Helper()
	r, e := Parse(page(fixture(id, body)), target)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestCollection(t *testing.T) {
	for _, raw := range []string{"urn:uuid:" + id1, "urn:uuid:" + strings.ToUpper(id1)} {
		v, e := Collection(raw)
		if e != nil || v != id1 {
			t.Fatalf("%q %s %v", raw, v, e)
		}
	}
	for _, raw := range []string{id1, "URN:UUID:" + id1, " urn:uuid:" + id1, "urn:uuid:" + id1 + " ", "urn:uuid:" + strings.ReplaceAll(id1, "-", ""), "urn:uuid:{" + id1 + "}", "urn:uuid:" + id1 + "#x", "https://collection.example/", "urn:uuid:xx"} {
		if _, e := Collection(raw); e == nil {
			t.Fatalf("受理 %q", raw)
		}
	}
}
func TestParseConstraints(t *testing.T) {
	valid := fixture(id1, "<p>一</p><p>二 &amp; 三</p>")
	cases := []struct {
		name, body string
		valid      bool
	}{
		{"normal", valid, true}, {"duplicate", strings.Replace(valid, "</head>", `<link rel="collection" href="urn:uuid:`+strings.ToUpper(id1)+`"></head>`, 1), true},
		{"multiple collection", strings.Replace(valid, "</head>", `<link rel="collection" href="urn:uuid:`+id2+`"></head>`, 1), false},
		{"invalid mixed", strings.Replace(valid, "</head>", `<link rel="collection" href="bad"></head>`, 1), false},
		{"whitespace urn", strings.Replace(valid, "urn:uuid:"+id1, " urn:uuid:"+id1, 1), false},
		{"no link", strings.Replace(valid, `href="`+target+`"`, `href="https://else.example/"`, 1), false},
		{"no entry", strings.Replace(valid, "h-entry", "article", 1), false},
		{"two entries", valid + `<div class="h-entry">second</div>`, false},
		{"h-feed entries", `<div class="h-feed">` + valid + `<div class="h-entry">second</div></div>`, false},
		{"nested entry", strings.Replace(valid, "</article>", `<div class="h-entry">nested</div></article>`, 1), false},
		{"relative base", strings.Replace(strings.Replace(valid, `href="`+target+`"`, `href="/"`, 1), "<head>", `<head><base href="https://relay.example/redirected">`, 1), true},
		{"relative final url", strings.Replace(valid, `href="`+target+`"`, `href="https://relay.example/"`, 1), true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r, e := Parse(page(tt.body), target)
			if (e == nil) != tt.valid {
				t.Fatalf("valid=%v error=%v", tt.valid, e)
			}
			if e == nil {
				var m map[string]any
				if e = json.Unmarshal(r.MF2, &m); e != nil {
					t.Fatal(e)
				}
				b, _ := json.Marshal(m)
				if !bytes.Equal(b, r.MF2) {
					t.Fatal("JSON不安定")
				}
				if strings.Contains(string(b), `"html"`) {
					t.Fatal("HTML残存")
				}
				r2, e := Parse(page(tt.body), target)
				if e != nil || r.Hash != r2.Hash {
					t.Fatal("不安定ハッシュ")
				}
			}
		})
	}
}
func TestLinkAttributes(t *testing.T) {
	for _, link := range []string{`<a href="%s">x</a>`, `<area href="%s">`, `<img src="%s">`, `<img href="%s">`, `<video src="%s"></video>`, `<audio src="%s"></audio>`, `<object data="%s"></object>`, `<link href="%s">`} {
		body := strings.Replace(fixture(id1, "ok"), `<a href="`+target+`">relay</a>`, fmt.Sprintf(link, target), 1)
		if _, e := Parse(page(body), target); e != nil {
			t.Fatalf("%s: %v", link, e)
		}
	}
	body := strings.Replace(fixture(id1, "ok"), `<a href="`+target+`">relay</a>`, `<p>`+target+`</p>`, 1)
	if _, e := Parse(page(body), target); e == nil {
		t.Fatal("テキストだけをHTMLリンク扱い")
	}
}
func TestConfig(t *testing.T) {
	c := config(t)
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{"", "ftp://relay.example/", "https://user@relay.example/", "https://relay.example/sub/", "https://relay.example/?x", "https://relay.example/#x", "https://relay.example/?"} {
		c.PublicURL = raw
		if c.Validate() == nil {
			t.Fatal(raw)
		}
	}
	c = config(t)
	c.Workers = 0
	if c.Validate() == nil {
		t.Fatal("無限worker")
	}
	t.Setenv("PUBLIC_URL", target)
	t.Setenv("DATA_DIR", c.DataDir)
	t.Setenv("MAX_BODY_BYTES", "0")
	if _, e := ConfigFromEnv(); e == nil {
		t.Fatal("無制限取得")
	}
}
func TestStorePersistenceAndTransactions(t *testing.T) {
	dir := t.TempDir()
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	j := Job{Source: "https://source.example/post", Target: target}
	r := mustRecord(t, id1, "first")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if e = s.Apply(ctx, j, j.Source, &r, now); e != nil {
		t.Fatal(e)
	}
	if e = s.Apply(ctx, j, j.Source, &r, now.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	up, ps, e := s.Feed(ctx, id1, 100)
	if e != nil || len(ps) != 1 || !up.Equal(now) || !ps[0].Updated.Equal(now) {
		t.Fatalf("同値更新 %v %v %v", up, ps, e)
	}
	if string(r.MF2) != ps[0].MF2 {
		t.Fatal("mf2読み戻し不一致")
	}
	s.Close()
	s, e = OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	_, ps, e = s.Feed(ctx, id1, 100)
	if e != nil || len(ps) != 1 {
		t.Fatal(e)
	}
	r2 := mustRecord(t, id2, "moved")
	if e = s.Apply(ctx, j, j.Source, &r2, now.Add(2*time.Hour)); e != nil {
		t.Fatal(e)
	}
	_, ps, e = s.Feed(ctx, id1, 100)
	if e != nil || len(ps) != 0 {
		t.Fatal("旧所属")
	}
	_, ps, e = s.Feed(ctx, id2, 100)
	if e != nil || len(ps) != 1 || !ps[0].Created.Equal(now) {
		t.Fatal("移動")
	}
	// DB側エラーでcollection日時と投稿の両方がrollbackされる。
	_, e = s.db.Exec(`CREATE TRIGGER fail_update BEFORE UPDATE ON posts BEGIN SELECT RAISE(ABORT,'test'); END`)
	if e != nil {
		t.Fatal(e)
	}
	r3 := mustRecord(t, id1, "rollback")
	if e = s.Apply(ctx, j, j.Source, &r3, now.Add(3*time.Hour)); e == nil {
		t.Fatal("rollback未発生")
	}
	up, ps, e = s.Feed(ctx, id1, 100)
	if e != nil || len(ps) != 0 || !up.Equal(now.Add(2*time.Hour)) {
		t.Fatal("部分反映")
	}
	s.db.Exec("DROP TRIGGER fail_update")
	if e = s.Apply(ctx, j, j.Source, nil, now.Add(4*time.Hour)); e != nil {
		t.Fatal(e)
	}
	_, ps, e = s.Feed(ctx, id2, 100)
	if e != nil || len(ps) != 0 {
		t.Fatal("空collection消失")
	}
}
func TestStoreSchemaAndUnavailable(t *testing.T) {
	dir := t.TempDir()
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	s.db.Exec("PRAGMA user_version=99")
	s.Close()
	if s, e = OpenStore(dir); e == nil {
		s.Close()
		t.Fatal("版不整合を受理")
	}
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, []byte("x"), 0600)
	if s, e = OpenStore(file); e == nil {
		s.Close()
		t.Fatal("書込不能を受理")
	}
	s = store(t)
	s.Close()
	if s.Health(context.Background()) == nil {
		t.Fatal("閉じたDBがhealthy")
	}
}
func TestQueue(t *testing.T) {
	s := store(t)
	ctx := context.Background()
	now := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := s.Enqueue(ctx, "https://source.example/post", target, 3, now)
			if e == nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			} else if !errors.Is(e, ErrFull) {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if accepted != 3 {
		t.Fatal(accepted)
	}
	j, e := s.Claim(ctx, now)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Claim(ctx, now); e != sql.ErrNoRows {
		t.Fatal("同一sourceを並列claim", e)
	}
	if e = s.Recover(ctx, 3); e != nil {
		t.Fatal(e)
	}
	j2, e := s.Claim(ctx, now)
	if e != nil || j2.ID != j.ID || j2.Attempts != 2 {
		t.Fatal("復旧", j2, e)
	}
	if e = s.Finish(ctx, j2, errors.New("temporary"), 3, now); e != nil {
		t.Fatal(e)
	}
	var state string
	var next int64
	s.db.QueryRow("SELECT state,next_at FROM jobs WHERE id=?", j.ID).Scan(&state, &next)
	if state != "pending" || next <= now.UnixNano() {
		t.Fatal("再試行遅延")
	}
	j2.Attempts = 3
	if e = s.Finish(ctx, j2, errors.New("temporary"), 3, now); e != nil {
		t.Fatal(e)
	}
	s.db.QueryRow("SELECT state FROM jobs WHERE id=?", j.ID).Scan(&state)
	if state != "failed" {
		t.Fatal(state)
	}
	if e = s.Cleanup(ctx, now.Add(8*24*time.Hour), 7*24*time.Hour); e != nil {
		t.Fatal(e)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM jobs").Scan(&n)
	if n != 0 {
		t.Fatal("保持期限", n)
	}
}

// ローカル固定sourceを読むテスト専用取得層。本番設定には存在しない。
type fixtureFetcher struct {
	client *http.Client
	base   string
}

func (f fixtureFetcher) Fetch(ctx context.Context, raw string) (Page, error) {
	if f.base != "" {
		raw = f.base + "/post"
	}
	r, e := http.NewRequestWithContext(ctx, "GET", raw, nil)
	if e != nil {
		return Page{}, e
	}
	res, e := f.client.Do(r)
	if e != nil {
		return Page{}, e
	}
	defer res.Body.Close()
	var b bytes.Buffer
	_, e = b.ReadFrom(res.Body)
	return Page{URL: res.Request.URL.String(), Status: res.StatusCode, Body: b.Bytes()}, e
}

type resultFetcher struct {
	p Page
	e error
}

func (f resultFetcher) Fetch(context.Context, string) (Page, error) { return f.p, f.e }
func TestProcessOutcomes(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name     string
		p        Page
		err      error
		withdraw bool
	}{
		{"404", Page{Status: 404}, nil, true}, {"410", Page{Status: 410}, nil, true}, {"link missing", page(strings.Replace(fixture(id1, "x"), target, "https://else.example/", 1)), nil, true}, {"qualification missing", page("<a href='" + target + "'>relay</a>"), nil, true}, {"5xx", Page{Status: 503}, nil, false}, {"timeout", Page{}, context.DeadlineExceeded, false}, {"safe refusal", Page{}, errors.New("unsafe source"), false}, {"size overflow", Page{}, errors.New("body limit"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := store(t)
			j := Job{Source: "https://source.example/post", Target: target}
			r := mustRecord(t, id1, "original")
			s.Apply(ctx, j, j.Source, &r, time.Now())
			a, _ := NewApp(config(t), s, resultFetcher{tt.p, tt.err})
			e := a.Process(ctx, j)
			_, posts, er := s.Feed(ctx, id1, 100)
			if er != nil {
				t.Fatal(er)
			}
			if tt.withdraw {
				if e != nil || len(posts) != 0 {
					t.Fatal(e, posts)
				}
			} else if e == nil || len(posts) != 1 {
				t.Fatal(e, posts)
			}
		})
	}
}
func TestHTTPAdmission(t *testing.T) {
	s := store(t)
	c := config(t)
	c.Queue = 1
	c.MaxForm = 300
	a, _ := NewApp(c, s, resultFetcher{})
	h := a.Handler()
	send := func(method, body string) int {
		r := httptest.NewRequest(method, "http://spoof.example/webmention", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	form := url.Values{"source": {"https://source.example/p"}, "target": {target}}.Encode()
	for _, body := range []string{"", url.Values{"source": {target}, "target": {target}}.Encode(), "source=ftp%3A%2F%2Fx&target=" + url.QueryEscape(target), form + "&source=x", form + strings.Repeat("x", 400)} {
		if n := send("POST", body); n != 400 {
			t.Fatal(n, body)
		}
	}
	if n := send("GET", form); n != 405 {
		t.Fatal(n)
	}
	if n := send("POST", form); n != 202 {
		t.Fatal(n)
	}
	if n := send("POST", form); n != 503 {
		t.Fatal(n)
	}
	s.Close()
	if n := send("POST", form); n == 202 {
		t.Fatal("非永続202")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://spoof.example/", nil))
	if !strings.Contains(w.Header().Get("Link"), target+"webmention") || !strings.Contains(w.Body.String(), `rel="webmention"`) {
		t.Fatal("広告")
	}
}
func TestAtom(t *testing.T) {
	r := mustRecord(t, id1, `&lt;script&gt;alert(1)&lt;/script&gt; &amp; tail`)
	p := Post{Source: "https://source.example/p", MF2: string(r.MF2), Updated: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	b, e := Atom(target, id1, p.Updated, []Post{p}, 100)
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		XMLName xml.Name
		ID      string `xml:"id"`
		Link    struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Entries []struct {
			ID      string `xml:"id"`
			Title   string `xml:"title"`
			Content struct {
				Type  string `xml:"type,attr"`
				Value string `xml:",chardata"`
			} `xml:"content"`
			Author struct {
				Name string `xml:"name"`
			} `xml:"author"`
		} `xml:"entry"`
	}
	if e = xml.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	if f.XMLName.Space != "http://www.w3.org/2005/Atom" || f.ID != "urn:uuid:"+id1 || len(f.Entries) != 1 || f.Entries[0].ID != p.Source || f.Entries[0].Content.Type != "text" || f.Entries[0].Content.Value != "<script>alert(1)</script> & tail" || f.Entries[0].Author.Name != "著者" {
		t.Fatalf("Atom不一致 %s", b)
	}
	if bytes.Contains(b, []byte("<script>")) {
		t.Fatal("XML内タグ")
	}
	b2, _ := Atom("https://other.example/", id1, p.Updated, []Post{p}, 100)
	if bytes.Equal(b, b2) || !bytes.Contains(b2, []byte("urn:uuid:"+id1)) {
		t.Fatal("別リレー")
	}
	fallback := Post{Source: p.Source, MF2: `{"type":["h-entry"],"properties":{}}`, Updated: p.Updated}
	b, _ = Atom(target, id1, p.Updated, []Post{fallback}, 100)
	if !bytes.Contains(b, []byte("投稿者不明")) || !bytes.Contains(b, []byte("<title>"+p.Source+"</title>")) {
		t.Fatal(string(b))
	}
}
func TestE2EAndRestart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	body := fixture(id1, "initial")
	status := 200
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	defer source.Close()
	c := config(t)
	c.Attempts = 1
	s, e := OpenStore(c.DataDir)
	if e != nil {
		t.Fatal(e)
	}
	a, _ := NewApp(c, s, fixtureFetcher{source.Client(), source.URL})
	if e = a.Start(ctx); e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(a.Handler())
	defer server.Close()
	notify := func() {
		t.Helper()
		form := url.Values{"source": {"https://source.example/post"}, "target": {target}}
		res, e := http.PostForm(server.URL+"/webmention", form)
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != 202 {
			t.Fatalf("受付 %d", res.StatusCode)
		}
	}
	wait := func(id string, count int, content string) []Post {
		t.Helper()
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			_, ps, e := s.Feed(ctx, id, 100)
			var n int
			s.db.QueryRow("SELECT count(*) FROM jobs WHERE state IN ('pending','running')").Scan(&n)
			if e == nil && len(ps) == count && n == 0 && (count == 0 || strings.Contains(ps[0].MF2, content)) {
				return ps
			}
			time.Sleep(15 * time.Millisecond)
		}
		t.Fatalf("処理未完 %s %d", id, count)
		return nil
	}
	notify()
	ps := wait(id1, 1, "initial")
	original := ps[0].Updated
	notify()
	ps = wait(id1, 1, "initial")
	if !ps[0].Updated.Equal(original) {
		t.Fatal("同値日時")
	}
	mu.Lock()
	body = fixture(id1, "edited")
	mu.Unlock()
	notify()
	wait(id1, 1, "edited")
	mu.Lock()
	body = fixture(id2, "moved")
	mu.Unlock()
	notify()
	wait(id2, 1, "moved")
	wait(id1, 0, "")
	mu.Lock()
	status = 503
	mu.Unlock()
	j := Job{Source: "https://source.example/post", Target: target}
	notify()
	wait(id2, 1, "moved")
	var failed int
	if e = s.db.QueryRow("SELECT count(*) FROM jobs WHERE state='failed'").Scan(&failed); e != nil || failed != 1 {
		t.Fatal("5xxの失敗状態", failed, e)
	}
	mu.Lock()
	status = 410
	mu.Unlock()
	notify()
	wait(id2, 0, "")
	for _, id := range []string{id1, id2} {
		res, e := http.Get(server.URL + "/collections/" + id + ".atom")
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Content-Type"), "application/atom+xml") {
			t.Fatal(res.StatusCode)
		}
		res.Body.Close()
	}
	res, _ := http.Get(server.URL + "/collections/550e8400-e29b-41d4-a716-446655449999.atom")
	if res.StatusCode != 404 {
		t.Fatal(res.StatusCode)
	}
	res.Body.Close()
	// 停止後に永続受付し、別のStore/Appで再起動する。
	cancel()
	a.Wait()
	mu.Lock()
	status = 200
	body = fixture(id1, "restart")
	mu.Unlock()
	if e = s.Enqueue(context.Background(), j.Source, target, 100, time.Now()); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = OpenStore(c.DataDir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	a2, _ := NewApp(c, s, fixtureFetcher{source.Client(), source.URL})
	if e = a2.Start(ctx2); e != nil {
		t.Fatal(e)
	}
	defer func() { cancel2(); a2.Wait() }()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		_, posts, e := s.Feed(ctx2, id1, 100)
		if e == nil && len(posts) == 1 && strings.Contains(posts[0].MF2, "restart") {
			if posts[0].Source != j.Source {
				t.Fatal("id変更")
			}
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("再起動通知未処理")
}
