package relay

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

var ErrFull = errors.New("通知キュー満杯")
var ErrStored = errors.New("保存済みsource")
var ErrUnknown = errors.New("未知のcollection")

type Store struct{ db *sql.DB }
type Post struct {
	Source, Target, FinalURL, Collection, MF2 string
	Created, Updated                          time.Time
}
type Job struct {
	ID             int64
	Source, Target string
	Attempts       int
}

func OpenStore(dir string) (*Store, error) {
	if e := os.MkdirAll(dir, 0750); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", filepath.Join(dir, "mezzanine.db"))
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	s := &Store{db}
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	if _, e = db.Exec("PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON"); e != nil {
		return fail(e)
	}
	var version int
	if e = db.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return fail(e)
	}
	if version != 0 && version != 2 {
		return fail(fmt.Errorf("非互換スキーマ版 %d", version))
	}
	if version == 0 {
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		defer tx.Rollback()
		_, e = tx.Exec(`CREATE TABLE collections(id TEXT PRIMARY KEY,updated TEXT NOT NULL);
CREATE TABLE posts(source TEXT NOT NULL,target TEXT NOT NULL,final_url TEXT NOT NULL,collection TEXT NOT NULL REFERENCES collections(id),mf2 TEXT NOT NULL,created TEXT NOT NULL,entry_sec INTEGER NOT NULL,entry_nano INTEGER NOT NULL,PRIMARY KEY(source));
CREATE TABLE jobs(id INTEGER PRIMARY KEY,source TEXT NOT NULL,target TEXT NOT NULL,state TEXT NOT NULL CHECK(state IN ('pending','running','done','failed')),attempts INTEGER NOT NULL DEFAULT 0,next_at INTEGER NOT NULL,created INTEGER NOT NULL,finished INTEGER,error TEXT NOT NULL DEFAULT '');
CREATE INDEX posts_feed ON posts(collection,entry_sec DESC,entry_nano DESC,source);
CREATE INDEX jobs_ready ON jobs(state,next_at,id); CREATE INDEX jobs_source ON jobs(source,state); PRAGMA user_version=2;`)
		if e != nil {
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}
	return s, nil
}
func (s *Store) Close() error  { return s.db.Close() }
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func (s *Store) Enqueue(ctx context.Context, source, target string, limit int, now time.Time) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var known bool
	if e = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE source=?)", source).Scan(&known); e != nil {
		return e
	}
	if known {
		return ErrStored
	}
	var n int
	e = tx.QueryRow("SELECT count(*) FROM jobs WHERE state IN ('pending','running')").Scan(&n)
	if e != nil {
		return e
	}
	if n >= limit {
		return ErrFull
	}
	_, e = tx.Exec("INSERT INTO jobs(source,target,state,next_at,created) VALUES(?,?,'pending',?,?)", source, target, now.UnixNano(), now.UnixNano())
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Recover(ctx context.Context, max int) error {
	_, e := s.db.ExecContext(ctx, "UPDATE jobs SET state=CASE WHEN attempts>=? AND NOT EXISTS(SELECT 1 FROM posts WHERE posts.source=jobs.source) THEN 'failed' ELSE 'pending' END, finished=CASE WHEN attempts>=? AND NOT EXISTS(SELECT 1 FROM posts WHERE posts.source=jobs.source) THEN ? ELSE NULL END WHERE state='running'", max, max, time.Now().UnixNano())
	return e
}
func (s *Store) Claim(ctx context.Context, now time.Time) (Job, error) {
	var j Job
	e := s.db.QueryRowContext(ctx, `UPDATE jobs SET state='running',attempts=attempts+1 WHERE id=(SELECT j.id FROM jobs j WHERE j.state='pending' AND j.next_at<=? AND NOT EXISTS(SELECT 1 FROM jobs r WHERE r.source=j.source AND r.state='running') ORDER BY j.id LIMIT 1) RETURNING id,source,target,attempts`, now.UnixNano()).Scan(&j.ID, &j.Source, &j.Target, &j.Attempts)
	return j, e
}
func (s *Store) Finish(ctx context.Context, j Job, cause error, max int, now time.Time) error {
	if cause == nil {
		_, e := s.db.ExecContext(ctx, "UPDATE jobs SET state='done',finished=?,error='' WHERE id=?", now.UnixNano(), j.ID)
		return e
	}
	// 原本・URL・認証情報をエラー本文へ永続化しない。
	if j.Attempts >= max || errors.Is(cause, ErrIneligible) {
		_, e := s.db.ExecContext(ctx, "UPDATE jobs SET state='failed',finished=?,error='processing failed' WHERE id=?", now.UnixNano(), j.ID)
		return e
	}
	_, e := s.db.ExecContext(ctx, "UPDATE jobs SET state='pending',next_at=?,error='processing failed' WHERE id=?", now.Add(time.Duration(j.Attempts)*time.Second).UnixNano(), j.ID)
	return e
}
func (s *Store) Cleanup(ctx context.Context, now time.Time, retention time.Duration) error {
	_, e := s.db.ExecContext(ctx, "DELETE FROM jobs WHERE (state IN ('done','failed') AND finished<?) OR (state='pending' AND created<?)", now.Add(-retention).UnixNano(), now.Add(-retention).UnixNano())
	return e
}
func (s *Store) HasSource(ctx context.Context, source string) (bool, error) {
	var known bool
	e := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM posts WHERE source=?)", source).Scan(&known)
	return known, e
}
func (s *Store) Apply(ctx context.Context, j Job, final string, r *Record, now time.Time) error {
	if r == nil {
		return ErrIneligible
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var known bool
	if e = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE source=?)", j.Source).Scan(&known); e != nil {
		return e
	}
	if known {
		return nil
	}
	if _, e = tx.Exec("INSERT INTO collections(id,updated) VALUES(?,?) ON CONFLICT(id) DO NOTHING", r.Collection, stamp(now)); e != nil {
		return e
	}
	entry := entryTime(Post{MF2: string(r.MF2), Updated: now})
	result, e := tx.Exec("INSERT INTO posts(source,target,final_url,collection,mf2,created,entry_sec,entry_nano) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(source) DO NOTHING", j.Source, j.Target, final, r.Collection, string(r.MF2), stamp(now), entry.Unix(), entry.Nanosecond())
	if e != nil {
		return e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return nil
	}
	if _, e = tx.Exec("UPDATE collections SET updated=? WHERE id=?", stamp(now), r.Collection); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Feed(ctx context.Context, id string, limit int) (time.Time, []Post, error) {
	// 同一読み取りトランザクションでcollection日時と投稿を揃える。
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return time.Time{}, nil, e
	}
	defer tx.Rollback()
	var updated string
	e = tx.QueryRow("SELECT updated FROM collections WHERE id=?", id).Scan(&updated)
	if e == sql.ErrNoRows {
		return time.Time{}, nil, ErrUnknown
	}
	if e != nil {
		return time.Time{}, nil, e
	}
	rows, e := tx.Query("SELECT source,target,final_url,collection,mf2,created FROM posts WHERE collection=? ORDER BY entry_sec DESC,entry_nano DESC,source ASC LIMIT ?", id, limit)
	if e != nil {
		return time.Time{}, nil, e
	}
	var posts []Post
	for rows.Next() {
		var p Post
		var cr string
		if e = rows.Scan(&p.Source, &p.Target, &p.FinalURL, &p.Collection, &p.MF2, &cr); e != nil {
			rows.Close()
			return time.Time{}, nil, e
		}
		p.Created, _ = time.Parse(time.RFC3339Nano, cr)
		p.Updated = p.Created
		posts = append(posts, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return time.Time{}, nil, e
	}
	if e = tx.Commit(); e != nil {
		return time.Time{}, nil, e
	}
	t, e := time.Parse(time.RFC3339Nano, updated)
	return t, posts, e
}
func (s *Store) Health(ctx context.Context) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec("UPDATE collections SET updated=updated WHERE id=(SELECT id FROM collections LIMIT 1)")
	return e
}
