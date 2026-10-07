package relay

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDateFallback(t *testing.T) {
	for _, raw := range []string{"2026-01-01", "2026-01-01T12:30", "2026-01-01T12:30:45", "2026-01-01T12:30:45+09:00"} {
		if _, e := parseDate(raw); e != nil {
			t.Fatal(raw, e)
		}
	}
	for _, raw := range []string{"", "2026-13-01", "yesterday"} {
		if _, e := parseDate(raw); e == nil {
			t.Fatal(raw)
		}
	}
	now := time.Now()
	p := Post{Source: "https://source.example/p", Updated: now, MF2: `{"properties":{"updated":["bad"],"published":["bad"]}}`}
	if !entryTime(p).Equal(now) {
		t.Fatal("不正updatedの代替")
	}
	b, e := Atom(target, id1, now, []Post{p}, 100)
	if e != nil || strings.Contains(string(b), "<published>") {
		t.Fatal(string(b), e)
	}
	c := config(t)
	c.PublicURL = "https://relay.example"
	if e = c.Validate(); e != nil {
		t.Fatal("パスなしのルートURL", e)
	}
}
func TestWorkerRetriesFinite(t *testing.T) {
	c := config(t)
	c.Attempts = 2
	s := store(t)
	a, _ := NewApp(c, s, resultFetcher{p: Page{Status: 503}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if e := a.Start(ctx); e != nil {
		t.Fatal(e)
	}
	defer func() { cancel(); a.Wait() }()
	if e := s.Enqueue(ctx, "https://source.example/retry", target, 100, time.Now()); e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		var state, errText string
		var attempts int
		if e := s.db.QueryRow("SELECT state,attempts,error FROM jobs").Scan(&state, &attempts, &errText); e != nil {
			t.Fatal(e)
		}
		if state == "failed" {
			if attempts != 2 || errText != "processing failed" {
				t.Fatal(state, attempts, errText)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("workerの再試行が終了しない")
}
