package relay

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSystemIdentity(t *testing.T) {
	s := store(t)
	a, err := NewApp(config(t), s, resultFetcher{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest("GET", target, nil))
	if w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	for _, want := range []string{"<title>indie-mezzanine</title>", "<h1>indie-mezzanine</h1>"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
	now := time.Now().UTC()
	record := mustRecord(t, id1, "本文")
	source := "https://source.example/post"
	data, err := Atom(target, id1, now, []Post{{Source: source, MF2: string(record.MF2), Updated: now}}, 100)
	if err != nil {
		t.Fatal(err)
	}
	var feed struct {
		Title  string `xml:"title"`
		ID     string `xml:"id"`
		Author struct {
			Name string `xml:"name"`
		} `xml:"author"`
		Entries []struct {
			ID     string `xml:"id"`
			Author struct {
				Name string `xml:"name"`
			} `xml:"author"`
		} `xml:"entry"`
	}
	if err = xml.Unmarshal(data, &feed); err != nil {
		t.Fatal(err)
	}
	if feed.Author.Name != "indie-mezzanine" || feed.ID != "urn:uuid:"+id1 || feed.Title != feed.ID {
		t.Fatalf("feed identity: %+v", feed)
	}
	if len(feed.Entries) != 1 || feed.Entries[0].ID != source || feed.Entries[0].Author.Name != "著者" {
		t.Fatalf("entry identity: %+v", feed.Entries)
	}
}

func TestFetcherSystemIdentity(t *testing.T) {
	f, err := NewSafeFetcher(config(t))
	if err != nil {
		t.Fatal(err)
	}
	defer f.client.CloseIdleConnections()
	f.client.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("User-Agent"); got != "indie-mezzanine Webmention receiver" {
			t.Errorf("User-Agent = %q", got)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/html"}}, Body: io.NopCloser(strings.NewReader("<p>fixture</p>")), Request: r}, nil
	})
	if _, err = f.Fetch(context.Background(), "https://source.example/post"); err != nil {
		t.Fatal(err)
	}
}

func TestStoreSystemIdentity(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	record := mustRecord(t, id1, "旧ファイル内の投稿")
	source := "https://source.example/old"
	if err = s.Apply(context.Background(), Job{Source: source, Target: target}, source, &record, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(dir, "indie-mezzanine.db")
	legacy := filepath.Join(dir, "mezzanine.db") // 旧名の非使用を検証するfixture。
	if _, err = os.Stat(current); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("unexpected legacy DB: %v", err)
	}
	if err = os.Rename(current, legacy); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(legacy)
	if err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, _, err = s.Feed(context.Background(), id1, 100); !errors.Is(err, ErrUnknown) {
		t.Fatalf("legacy data read: %v", err)
	}
	if _, err = os.Stat(current); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("legacy file changed")
	}
}
