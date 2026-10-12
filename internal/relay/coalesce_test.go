package relay

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type jobSnapshot struct {
	id, attempts, next, created      int64
	source, target, state, errorText string
}

func TestCoalesceHTTP(t *testing.T) {
	ctx := context.Background()
	c, s := config(t), store(t)
	defer s.Close()
	c.Queue = 2
	f := &countFetcher{}
	a, _ := NewApp(c, s, f)
	source := "https://source.example/p"
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if w := rootNotify(a, source, target); w.Code != 202 {
				t.Error(w.Code)
			}
		}()
	}
	wg.Wait()
	before := readJob(t, s, source)
	if w := rootNotify(a, source, target); w.Code != 202 || !strings.Contains(w.Body.String(), "既存の処理を継続") {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := rootNotify(a, source, "https://other.example/"); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if after := readJob(t, s, source); after != before || jobCount(t, s) != 1 || f.calls != 0 {
		t.Fatal(after)
	}
	if w := rootNotify(a, "https://source.example/other", target); w.Code != 202 {
		t.Fatal(w.Code)
	}
	if w := rootNotify(a, source, target); w.Code != 202 {
		t.Fatal(w.Code)
	}
	if w := rootNotify(a, "https://source.example/full", target); w.Code != 503 {
		t.Fatal(w.Code)
	}
	j, e := s.Claim(ctx, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	r := mustRecord(t, id1, "original")
	if e := s.Apply(ctx, j, source, &r, time.Now()); e != nil {
		t.Fatal(e)
	}
	_, posts, e := s.Feed(ctx, id1, 10)
	if e != nil || len(posts) != 1 {
		t.Fatal(e)
	}
	if w := rootNotify(a, source, target); w.Code != 200 {
		t.Fatal(w.Code)
	}
	_, after, e := s.Feed(ctx, id1, 10)
	if e != nil || after[0] != posts[0] || jobCount(t, s) != 2 || f.calls != 0 {
		t.Fatal(e)
	}
	s.Close()
	if w := rootNotify(a, "https://source.example/other", target); w.Code != 503 {
		t.Fatal(w.Code)
	}
}

type coalesceFetcher struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
	p       Page
}

func (f *coalesceFetcher) Fetch(ctx context.Context, source string) (Page, error) {
	f.calls.Add(1)
	close(f.started)
	select {
	case <-f.release:
		return f.p, nil
	case <-ctx.Done():
		return Page{}, ctx.Err()
	}
}

func TestCoalesceDuringProcessing(t *testing.T) {
	for _, status := range []int{200, 404} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			ctx := context.Background()
			c, s := config(t), store(t)
			defer s.Close()
			f := &coalesceFetcher{started: make(chan struct{}), release: make(chan struct{}), p: page(fixture(id1, "ok"))}
			f.p.Status = status
			a, _ := NewApp(c, s, f)
			source := "https://source.example/p"
			for i := 0; i < 3; i++ {
				if w := rootNotify(a, source, target); w.Code != 202 {
					t.Fatal(w.Code)
				}
			}
			j, e := s.Claim(ctx, time.Now())
			if e != nil {
				t.Fatal(e)
			}
			before := readJob(t, s, source)
			processCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- a.Process(processCtx, j) }()
			select {
			case <-f.started:
			case <-time.After(5 * time.Second):
				t.Fatal("fetch did not start")
			}
			for i := 0; i < 3; i++ {
				if w := rootNotify(a, source, target); w.Code != 202 {
					t.Fatal(w.Code)
				}
			}
			if after := readJob(t, s, source); after != before {
				t.Fatal(after)
			}
			close(f.release)
			cause := <-result
			if status == 200 && cause != nil {
				t.Fatal(cause)
			}
			if status == 404 && !errors.Is(cause, ErrIneligible) {
				t.Fatal(cause)
			}
			if e := s.Finish(ctx, j, cause, c.Attempts, time.Now()); e != nil {
				t.Fatal(e)
			}
			if f.calls.Load() != 1 || jobCount(t, s) != 1 {
				t.Fatal(f.calls.Load())
			}
			if _, e := s.Claim(ctx, time.Now()); e != sql.ErrNoRows {
				t.Fatal(e)
			}
			known, e := s.HasSource(ctx, source)
			if e != nil || known != (status == 200) {
				t.Fatal(known, e)
			}
		})
	}
}

func TestCoalesceRetryAndRecovery(t *testing.T) {
	ctx := context.Background()
	s := store(t)
	defer s.Close()
	now := time.Now()
	source := "https://source.example/p"
	if e := s.Enqueue(ctx, source, target, 1, now); e != nil {
		t.Fatal(e)
	}
	j, e := s.Claim(ctx, now)
	if e != nil {
		t.Fatal(e)
	}
	if e := s.Recover(ctx, 3); e != nil {
		t.Fatal(e)
	}
	before := readJob(t, s, source)
	if e := s.Enqueue(ctx, source, target, 1, now.Add(time.Hour)); !errors.Is(e, ErrOutstanding) {
		t.Fatal(e)
	}
	if readJob(t, s, source) != before {
		t.Fatal("recovery job changed")
	}
	j, e = s.Claim(ctx, now)
	if e != nil || j.Attempts != 2 {
		t.Fatal(j, e)
	}
	if e := s.Finish(ctx, j, errors.New("temporary"), 3, now); e != nil {
		t.Fatal(e)
	}
	before = readJob(t, s, source)
	if e := s.Enqueue(ctx, source, target, 1, now.Add(time.Hour)); !errors.Is(e, ErrOutstanding) {
		t.Fatal(e)
	}
	if readJob(t, s, source) != before {
		t.Fatal("retry schedule changed")
	}
	if _, e := s.Claim(ctx, now); e != sql.ErrNoRows {
		t.Fatal(e)
	}
	j, e = s.Claim(ctx, now.Add(3*time.Second))
	if e != nil || j.Attempts != 3 {
		t.Fatal(j, e)
	}
	if e := s.Finish(ctx, j, errors.New("temporary"), 3, now); e != nil {
		t.Fatal(e)
	}
	if readJob(t, s, source).state != "failed" || jobCount(t, s) != 1 {
		t.Fatal("finite retry changed")
	}
}

func TestCoalesceRenotifyAfterFailureAndCleanup(t *testing.T) {
	ctx := context.Background()
	c, s := config(t), store(t)
	defer s.Close()
	c.Queue = 1
	a, _ := NewApp(c, s, resultFetcher{})
	now := time.Now()
	source := "https://source.example/p"
	if w := rootNotify(a, source, target); w.Code != 202 {
		t.Fatal(w.Code)
	}
	j, e := s.Claim(ctx, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if e := s.Finish(ctx, j, ErrIneligible, 3, now); e != nil {
		t.Fatal(e)
	}
	if w := rootNotify(a, "https://source.example/other", target); w.Code != 202 {
		t.Fatal(w.Code)
	}
	if w := rootNotify(a, source, target); w.Code != 503 {
		t.Fatal(w.Code)
	}
	if e := s.Cleanup(ctx, now.Add(8*24*time.Hour), 7*24*time.Hour); e != nil {
		t.Fatal(e)
	}
	if jobCount(t, s) != 0 {
		t.Fatal("expired jobs retained")
	}
	if w := rootNotify(a, source, target); w.Code != 202 {
		t.Fatal(w.Code)
	}
	if got := readJob(t, s, source); got.attempts != 0 || got.state != "pending" {
		t.Fatal(got)
	}
	j, e = s.Claim(ctx, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	r := mustRecord(t, id1, "original")
	if e := s.Apply(ctx, j, source, &r, now); e != nil {
		t.Fatal(e)
	}
	if e := s.Finish(ctx, j, nil, 3, now); e != nil {
		t.Fatal(e)
	}
	if e := s.Cleanup(ctx, now.Add(8*24*time.Hour), 7*24*time.Hour); e != nil {
		t.Fatal(e)
	}
	if w := rootNotify(a, source, target); w.Code != 200 || jobCount(t, s) != 0 {
		t.Fatal(w.Code)
	}
	_, posts, e := s.Feed(ctx, id1, 10)
	if e != nil || len(posts) != 1 || posts[0].MF2 != string(r.MF2) {
		t.Fatal(posts, e)
	}
}

func readJob(t *testing.T, s *Store, source string) jobSnapshot {
	t.Helper()
	var j jobSnapshot
	e := s.db.QueryRow("SELECT id,source,target,state,attempts,next_at,created,error FROM jobs WHERE source=? ORDER BY id DESC LIMIT 1", source).Scan(&j.id, &j.source, &j.target, &j.state, &j.attempts, &j.next, &j.created, &j.errorText)
	if e != nil {
		t.Fatal(e)
	}
	return j
}

func jobCount(t *testing.T, s *Store) int {
	t.Helper()
	var n int
	if e := s.db.QueryRow("SELECT count(*) FROM jobs").Scan(&n); e != nil {
		t.Fatal(e)
	}
	return n
}

func TestCoalesceStore(t *testing.T) {
	ctx := context.Background()
	for _, state := range []string{"pending", "running"} {
		t.Run(state, func(t *testing.T) {
			s := store(t)
			defer s.Close()
			now := time.Now()
			source := "https://source.example/p"
			if e := s.Enqueue(ctx, source, target, 2, now); e != nil {
				t.Fatal(e)
			}
			if state == "running" {
				if _, e := s.Claim(ctx, now); e != nil {
					t.Fatal(e)
				}
			}
			before := readJob(t, s, source)
			var wg sync.WaitGroup
			for i := 0; i < 16; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if e := s.Enqueue(ctx, source, "https://other.example/", 2, now.Add(time.Hour)); !errors.Is(e, ErrOutstanding) {
						t.Error(e)
					}
				}()
			}
			wg.Wait()
			if after := readJob(t, s, source); after != before || jobCount(t, s) != 1 {
				t.Fatal(before, after)
			}
			if e := s.Enqueue(ctx, "https://source.example/other", target, 2, now); e != nil {
				t.Fatal(e)
			}
			if e := s.Enqueue(ctx, source, target, 2, now); !errors.Is(e, ErrOutstanding) {
				t.Fatal(e)
			}
			if e := s.Enqueue(ctx, "https://source.example/full", target, 2, now); !errors.Is(e, ErrFull) {
				t.Fatal(e)
			}
			if jobCount(t, s) != 2 {
				t.Fatal("duplicate consumed capacity")
			}
		})
	}
	for _, state := range []string{"done", "failed"} {
		t.Run(state, func(t *testing.T) {
			s := store(t)
			defer s.Close()
			now := time.Now()
			source := "https://source.example/p"
			if e := s.Enqueue(ctx, source, target, 1, now); e != nil {
				t.Fatal(e)
			}
			j, e := s.Claim(ctx, now)
			if e != nil {
				t.Fatal(e)
			}
			var cause error
			if state == "failed" {
				cause = ErrIneligible
			}
			if e := s.Finish(ctx, j, cause, 3, now); e != nil {
				t.Fatal(e)
			}
			if e := s.Enqueue(ctx, source, target, 1, now); e != nil {
				t.Fatal(e)
			}
			if got := readJob(t, s, source); got.id == j.ID || got.attempts != 0 || got.state != "pending" {
				t.Fatal(got)
			}
		})
	}
}
