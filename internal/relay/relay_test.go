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
				if e != nil || !bytes.Equal(r.MF2, r2.MF2) {
					t.Fatal("不安定JSON")
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
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	changed := mustRecord(t, id2, "edited and moved")
	other := j
	other.Target = "https://other.example/"
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := s.Apply(ctx, other, "https://changed.example/", &changed, now.Add(time.Hour)); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	up, ps, e := s.Feed(ctx, id1, 100)
	if e != nil || len(ps) != 1 || !up.Equal(now) || !ps[0].Updated.Equal(now) || !ps[0].Created.Equal(now) || ps[0].MF2 != string(r.MF2) || ps[0].Target != target || ps[0].FinalURL != j.Source {
		t.Fatalf("スナップショット変動: %v %v %v", up, ps, e)
	}
	if _, _, e = s.Feed(ctx, id2, 100); e != ErrUnknown {
		t.Fatal("再通知で新collection作成", e)
	}
	if _, e = s.db.Exec("INSERT INTO posts SELECT source, 'https://other.example/', final_url, collection, mf2, created, entry_sec, entry_nano FROM posts"); e == nil {
		t.Fatal("一意制約なし")
	}
	if _, e = s.db.Exec(`CREATE TRIGGER fail_insert BEFORE INSERT ON posts BEGIN SELECT RAISE(ABORT,'test'); END`); e != nil {
		t.Fatal(e)
	}
	other.Source = "https://source.example/new"
	if e = s.Apply(ctx, other, other.Source, &changed, now.Add(2*time.Hour)); e == nil {
		t.Fatal("rollback未発生")
	}
	if _, _, e = s.Feed(ctx, id2, 100); e != ErrUnknown {
		t.Fatal("collection部分反映", e)
	}
	up, ps, e = s.Feed(ctx, id1, 100)
	if e != nil || len(ps) != 1 || !up.Equal(now) {
		t.Fatal("既存データ変動", e)
	}
}

func TestStoreSchemaAndUnavailable(t *testing.T) {
	dir := t.TempDir()
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	s.db.Exec("PRAGMA user_version=1")
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
		terminal bool
	}{
		{"404", Page{Status: 404}, nil, true}, {"410", Page{Status: 410}, nil, true},
		{"link missing", page(strings.Replace(fixture(id1, "x"), target, "https://else.example/", 1)), nil, true},
		{"qualification missing", page("<a href='" + target + "'>relay</a>"), nil, true},
		{"5xx", Page{Status: 503}, nil, false}, {"timeout", Page{}, context.DeadlineExceeded, false},
		{"safe refusal", Page{}, errors.New("unsafe source"), false}, {"size overflow", Page{}, errors.New("body limit"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := store(t)
			j := Job{Source: "https://source.example/post", Target: target}
			f := &countFetcher{p: tt.p, err: tt.err}
			a, _ := NewApp(config(t), s, f)
			e := a.Process(ctx, j)
			if e == nil || errors.Is(e, ErrIneligible) != tt.terminal {
				t.Fatal("初回失敗判定", e)
			}
			known, e := s.HasSource(ctx, j.Source)
			if e != nil || known {
				t.Fatal("失敗を保存", e)
			}
			r := mustRecord(t, id1, "original")
			now := time.Now()
			if e = s.Apply(ctx, j, j.Source, &r, now); e != nil {
				t.Fatal(e)
			}
			f.calls = 0
			if e = a.Process(ctx, j); e != nil || f.calls != 0 {
				t.Fatal("保存済みを再取得", f.calls, e)
			}
			up, ps, e := s.Feed(ctx, id1, 100)
			if e != nil || len(ps) != 1 || ps[0].MF2 != string(r.MF2) || !up.Equal(now) {
				t.Fatal("保存後の不変性", ps, e)
			}
		})
	}
}

type countFetcher struct {
	p     Page
	err   error
	calls int
}

func (f *countFetcher) Fetch(context.Context, string) (Page, error) { f.calls++; return f.p, f.err }

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
	r := mustRecord(t, id1, "stored")
	if e := s.Apply(context.Background(), Job{Source: "https://source.example/p", Target: target}, "https://source.example/p", &r, time.Now()); e != nil {
		t.Fatal(e)
	}
	if n := send("POST", form); n != 200 {
		t.Fatal("満杯時の保存済みsource", n)
	}
	var jobs int
	if e := s.db.QueryRow("SELECT count(*) FROM jobs").Scan(&jobs); e != nil || jobs != 1 {
		t.Fatal("job追加", jobs, e)
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
	status := 404
	calls := 0
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls++
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
	defer func() { cancel(); a.Wait() }()
	server := httptest.NewServer(a.Handler())
	defer server.Close()
	notify := func(raw string, want int) {
		t.Helper()
		res, e := http.PostForm(server.URL+"/webmention", url.Values{"source": {raw}, "target": {target}})
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("受付 %d want %d", res.StatusCode, want)
		}
	}
	drain := func() {
		t.Helper()
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			var n int
			if e := s.db.QueryRow("SELECT count(*) FROM jobs WHERE state IN ('pending','running')").Scan(&n); e != nil {
				t.Fatal(e)
			}
			if n == 0 {
				return
			}
			time.Sleep(15 * time.Millisecond)
		}
		t.Fatal("job未完了")
	}
	raw := "https://source.example/post"
	notify(raw, 202)
	drain()
	if known, e := s.HasSource(ctx, raw); e != nil || known {
		t.Fatal("初回失敗を保存", e)
	}
	mu.Lock()
	status = 200
	mu.Unlock()
	notify(raw, 202)
	drain()
	up, ps, e := s.Feed(ctx, id1, 100)
	if e != nil || len(ps) != 1 {
		t.Fatal(ps, e)
	}
	original := ps[0]
	before, e := Atom(target, id1, up, ps, 100)
	if e != nil {
		t.Fatal(e)
	}
	for _, change := range []struct {
		body   string
		status int
	}{{fixture(id1, "edited"), 200}, {fixture(id2, "moved"), 200}, {"gone", 410}, {"missing", 404}, {"no link", 200}, {"unavailable", 503}} {
		mu.Lock()
		body = change.body
		status = change.status
		prev := calls
		mu.Unlock()
		notify(raw, 200)
		drain()
		mu.Lock()
		got := calls
		mu.Unlock()
		if got != prev {
			t.Fatal("再通知で取得", got, prev)
		}
		up, ps, e = s.Feed(ctx, id1, 100)
		if e != nil {
			t.Fatal(e)
		}
		after, e := Atom(target, id1, up, ps, 100)
		if e != nil || !bytes.Equal(before, after) {
			t.Fatal("Atom変動", e)
		}
	}
	if _, _, e = s.Feed(ctx, id2, 100); e != ErrUnknown {
		t.Fatal("所属移動", e)
	}
	res, e := http.Get(server.URL + "/collections/" + id1 + ".atom")
	if e != nil {
		t.Fatal(e)
	}
	if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Content-Type"), "application/atom+xml") {
		t.Fatal(res.StatusCode)
	}
	res.Body.Close()
	res, e = http.Get(server.URL + "/collections/" + id2 + ".atom")
	if e != nil {
		t.Fatal(e)
	}
	if res.StatusCode != 404 {
		t.Fatal(res.StatusCode)
	}
	res.Body.Close()
	cancel()
	a.Wait()
	// 受付済みjobが保存直後に中断したケースと、未掲載の通知を同じDBに残す。
	now := time.Now()
	if _, e = s.db.Exec("INSERT INTO jobs(source,target,state,attempts,next_at,created) VALUES(?,?,'running',3,?,?)", raw, target, now.UnixNano(), now.UnixNano()); e != nil {
		t.Fatal(e)
	}
	fresh := "https://source.example/new"
	if e = s.Enqueue(context.Background(), fresh, target, 100, now); e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = OpenStore(c.DataDir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	mu.Lock()
	body = fixture(id2, "restart")
	status = 200
	prior := calls
	mu.Unlock()
	ctx2, cancel2 := context.WithCancel(context.Background())
	a2, _ := NewApp(c, s, fixtureFetcher{source.Client(), source.URL})
	if e = a2.Start(ctx2); e != nil {
		t.Fatal(e)
	}
	defer func() { cancel2(); a2.Wait() }()
	drain()
	mu.Lock()
	got := calls
	mu.Unlock()
	if got != prior+1 {
		t.Fatal("保存済みjobを再取得", got, prior)
	}
	up, ps, e = s.Feed(ctx2, id1, 100)
	if e != nil || len(ps) != 1 || ps[0] != original {
		t.Fatal("復旧でスナップショット変動", ps, e)
	}
	after, e := Atom(target, id1, up, ps, 100)
	if e != nil || !bytes.Equal(before, after) {
		t.Fatal("復旧でAtom変動", e)
	}
	_, ps, e = s.Feed(ctx2, id2, 100)
	if e != nil || len(ps) != 1 || ps[0].Source != fresh || !strings.Contains(ps[0].MF2, "restart") {
		t.Fatal("未掲載通知の復旧", ps, e)
	}
	var state string
	if e = s.db.QueryRow("SELECT state FROM jobs WHERE source=? AND attempts>1", raw).Scan(&state); e != nil || state != "done" {
		t.Fatal("保存後中断job完了", state, e)
	}
}

func TestConcurrentFirstSnapshot(t *testing.T) {
	s := store(t)
	ctx := context.Background()
	r1 := mustRecord(t, id1, "one")
	r2 := mustRecord(t, id2, "two")
	j := Job{Source: "https://source.example/concurrent", Target: target}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := r1
			if i%2 == 1 {
				r = r2
			}
			if e := s.Apply(ctx, j, j.Source, &r, time.Now()); e != nil {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	var count int
	if e := s.db.QueryRow("SELECT count(*) FROM posts").Scan(&count); e != nil || count != 1 {
		t.Fatal("重複投稿", count, e)
	}
	if e := s.db.QueryRow("SELECT count(*) FROM collections").Scan(&count); e != nil || count != 1 {
		t.Fatal("競合で余計なcollection", count, e)
	}
	if e := s.Enqueue(ctx, j.Source, "https://other.example/", 0, time.Now()); !errors.Is(e, ErrStored) {
		t.Fatal("保存済みsourceの受付", e)
	}
}
func TestPermanentInitialFailure(t *testing.T) {
	s := store(t)
	ctx := context.Background()
	now := time.Now()
	if e := s.Enqueue(ctx, "https://source.example/missing", target, 100, now); e != nil {
		t.Fatal(e)
	}
	j, e := s.Claim(ctx, now)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Finish(ctx, j, ErrIneligible, 3, now); e != nil {
		t.Fatal(e)
	}
	var state string
	if e = s.db.QueryRow("SELECT state FROM jobs WHERE id=?", j.ID).Scan(&state); e != nil || state != "failed" {
		t.Fatal("検証失敗の再試行", state, e)
	}
}
