package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestMF2StructureAndStableJSON(t *testing.T) {
	body := strings.Replace(fixture(id1, "<b>内容</b>"), "</article>", `<div class="h-cite"><span class="p-name">子項目</span><div class="e-content"><b>子内容</b></div></div></article>`, 1)
	r, e := Parse(page(body), target)
	if e != nil {
		t.Fatal(e)
	}
	var m map[string]any
	json.Unmarshal(r.MF2, &m)
	children, _ := m["children"].([]any)
	if len(children) != 1 {
		t.Fatalf("children消失 %s", r.MF2)
	}
	p := m["properties"].(map[string]any)
	author, _ := first(p, "author").(map[string]any)
	if author == nil || author["properties"] == nil {
		t.Fatal("入れ子著者消失")
	}
	if text(first(p, "content")) != "内容" {
		t.Fatal("value消失")
	}
	other := strings.ReplaceAll(body, "<b>", "<i>")
	other = strings.ReplaceAll(other, "</b>", "</i>")
	r2, e := Parse(page(other), target)
	if e != nil || !bytes.Equal(r.MF2, r2.MF2) {
		t.Fatal("HTMLだけの変化で保存JSON変動", e)
	}
	// 最終取得URLの相対参照を使い、余計な外部URLを取得しない。
	ppage := page(strings.Replace(fixture(id1, "x"), `href="`+target+`"`, `href="../"`, 1))
	ppage.URL = "https://relay.example/redirected/post"
	if _, e = Parse(ppage, target); e != nil {
		t.Fatal("最終URLによる参照解決", e)
	}
}
func TestFeedOrderingLimitAndTimes(t *testing.T) {
	s := store(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range []string{"c", "b", "a"} {
		r := mustRecord(t, id1, name)
		if e := s.Apply(ctx, Job{Source: "https://source.example/" + name, Target: target}, "https://source.example/"+name, &r, now); e != nil {
			t.Fatal(e)
		}
	}
	up, ps, e := s.Feed(ctx, id1, 2)
	if e != nil || len(ps) != 2 || !strings.HasSuffix(ps[0].Source, "/a") || !strings.HasSuffix(ps[1].Source, "/b") {
		t.Fatal("SQLの順序/上限", ps, e)
	}
	b, e := Atom(target, id1, up, ps, 1)
	if e != nil {
		t.Fatal(e)
	}
	var feed struct {
		Entries []struct {
			ID string `xml:"id"`
		} `xml:"entry"`
	}
	if e = xml.Unmarshal(b, &feed); e != nil || len(feed.Entries) != 1 || !strings.HasSuffix(feed.Entries[0].ID, "/a") {
		t.Fatal(string(b), e)
	}
	b2, _ := Atom(target, id1, up, ps, 1)
	if string(b) != string(b2) {
		t.Fatal("再取得の不安定出力")
	}
	p := Post{Source: "https://source.example/new", Updated: now, MF2: `{"properties":{"updated":["2026-02-01T01:00:00+09:00"],"published":["2025-12-31T12:00:00Z"]}}`}
	b, _ = Atom(target, id1, up, []Post{p}, 100)
	if !strings.Contains(string(b), "2026-01-31T16:00:00Z") || !strings.Contains(string(b), "<published>2025-12-31T12:00:00Z</published>") {
		t.Fatal(string(b))
	}
}
func TestExampleAndDocumentCommands(t *testing.T) {
	html, e := os.ReadFile("../../examples/post.html")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Parse(page(string(html)), target); e != nil {
		t.Fatal("投稿例", e)
	}
	doc, e := os.ReadFile("../../docs/PUBLISHING.md")
	if e != nil {
		t.Fatal(e)
	}
	command := regexp.MustCompile("(?s)```sh\\n(curl .*?)\\n```").FindSubmatch(doc)
	if len(command) != 2 {
		t.Fatal("通知コマンドなし")
	}
	if _, e = exec.LookPath("curl"); e != nil {
		t.Fatal("文書コマンド検証にはcurlが必要")
	}
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, string(html)) }))
	defer source.Close()
	c := config(t)
	s := store(t)
	a, _ := NewApp(c, s, fixtureFetcher{source.Client(), source.URL})
	server := httptest.NewServer(a.Handler())
	defer server.Close()
	// worker開始前に文書のcurlで通知する。202と未掲載を区別する。
	cmd := exec.Command("sh", "-c", string(command[1]))
	cmd.Env = append(os.Environ(), "SOURCE=https://author.example/post", "PUBLIC_URL="+target, "WEBMENTION_ENDPOINT="+server.URL+"/webmention")
	out, e := cmd.CombinedOutput()
	if e != nil || !strings.Contains(string(out), "202 Accepted") {
		t.Fatalf("curl %v %s", e, out)
	}
	if _, _, e = s.Feed(context.Background(), id1, 100); e != ErrUnknown {
		t.Fatal("検証前に掲載")
	}
	ctx, cancel := context.WithCancel(context.Background())
	if e = a.Start(ctx); e != nil {
		t.Fatal(e)
	}
	defer func() { cancel(); a.Wait() }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		url := server.URL + "/collections/" + id1 + ".atom"
		res, e := http.Get(url)
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode == 200 {
			repeated := exec.Command("sh", "-c", string(command[1]))
			repeated.Env = cmd.Env
			out, e := repeated.CombinedOutput()
			if e != nil || !strings.Contains(string(out), "200 OK") {
				t.Fatalf("再通知curl %v %s", e, out)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("文書購読経路の掲載失敗")
}
func TestBackupRestoreAndInterruptedJobs(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	j := Job{Source: "https://source.example/post", Target: target}
	r := mustRecord(t, id1, "stored")
	if e = s.Enqueue(ctx, j.Source, target, 100, time.Now()); e != nil {
		t.Fatal(e)
	}
	claimed, e := s.Claim(ctx, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Apply(ctx, j, j.Source, &r, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	backup, e := os.ReadFile(filepath.Join(dir, "mezzanine.db"))
	if e != nil {
		t.Fatal(e)
	}
	restore := t.TempDir()
	if e = os.WriteFile(filepath.Join(restore, "mezzanine.db"), backup, 0600); e != nil {
		t.Fatal(e)
	}
	s, e = OpenStore(restore)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Recover(ctx, 3); e != nil {
		t.Fatal(e)
	}
	again, e := s.Claim(ctx, time.Now())
	if e != nil || again.ID != claimed.ID {
		t.Fatal("中断job復旧", again, e)
	}
	f := &countFetcher{}
	a, _ := NewApp(config(t), s, f)
	if e = a.Process(ctx, again); e != nil || f.calls != 0 {
		t.Fatal("復元済みsourceを取得", f.calls, e)
	}
	if e = s.Finish(ctx, again, nil, 3, time.Now()); e != nil {
		t.Fatal(e)
	}
	_, posts, e := s.Feed(ctx, id1, 100)
	if e != nil || len(posts) != 1 || posts[0].MF2 != string(r.MF2) || posts[0].Source != j.Source {
		t.Fatal("復元データ不一致", posts, e)
	}
	if _, e = s.db.Exec("INSERT INTO posts SELECT * FROM posts"); e == nil {
		t.Fatal("一意制約なし")
	}
}
func TestHealthAndShutdown(t *testing.T) {
	s := store(t)
	c := config(t)
	a, _ := NewApp(c, s, resultFetcher{})
	ctx, cancel := context.WithCancel(context.Background())
	if e := a.Start(ctx); e != nil {
		t.Fatal(e)
	}
	check := func() int {
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, httptest.NewRequest("GET", "http://local/healthz", nil))
		return w.Code
	}
	if n := check(); n != 200 {
		t.Fatal(n)
	}
	server := a.Server()
	if server.ReadTimeout <= 0 || server.WriteTimeout <= 0 || server.ReadHeaderTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatal("無限HTTP timeout")
	}
	cancel()
	a.Wait()
	if n := check(); n != 503 {
		t.Fatal("worker停止後healthy", n)
	}
	a.active.Store(int32(c.Workers))
	s.Close()
	if n := check(); n != 503 {
		t.Fatal("保存先異常がhealthy", n)
	}
}
