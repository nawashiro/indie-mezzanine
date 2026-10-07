package relay

import (
	"encoding/json"
	"encoding/xml"
	"github.com/gorilla/feeds"
	"sort"
	"strings"
	"time"
)

func properties(raw string) map[string]any {
	var m map[string]any
	_ = json.Unmarshal([]byte(raw), &m)
	p, _ := m["properties"].(map[string]any)
	return p
}
func first(p map[string]any, k string) any {
	v, _ := p[k].([]any)
	if len(v) > 0 {
		return v[0]
	}
	return nil
}
func text(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case map[string]any:
		if s, ok := x["value"].(string); ok {
			return s
		}
	}
	return ""
}
func parseDate(raw string) (time.Time, error) {
	var last error
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"} {
		t, e := time.Parse(layout, raw)
		if e == nil {
			return t, nil
		}
		last = e
	}
	return time.Time{}, last
}
func entryTime(p Post) time.Time {
	if t, e := parseDate(text(first(properties(p.MF2), "updated"))); e == nil {
		return t
	}
	return p.Updated
}
func Atom(public, id string, updated time.Time, posts []Post, limit int) ([]byte, error) {
	sort.Slice(posts, func(i, j int) bool {
		a, b := entryTime(posts[i]), entryTime(posts[j])
		if a.Equal(b) {
			return posts[i].Source < posts[j].Source
		}
		return a.After(b)
	})
	if len(posts) > limit {
		posts = posts[:limit]
	}
	f := feeds.AtomFeed{Xmlns: "http://www.w3.org/2005/Atom", Title: "urn:uuid:" + id, Id: "urn:uuid:" + id, Updated: updated.UTC().Format(time.RFC3339Nano), Link: &feeds.AtomLink{Href: strings.TrimRight(public, "/") + "/collections/" + id + ".atom", Rel: "self", Type: "application/atom+xml"}, Author: &feeds.AtomAuthor{Name: "mezzanine relay"}}
	for _, p := range posts {
		props := properties(p.MF2)
		title := text(first(props, "name"))
		if title == "" {
			title = p.Source
		}
		author := "投稿者不明"
		if a, ok := first(props, "author").(map[string]any); ok {
			if ap, ok := a["properties"].(map[string]any); ok {
				if n := text(first(ap, "name")); n != "" {
					author = n
				}
			}
		}
		e := &feeds.AtomEntry{Title: title, Id: p.Source, Updated: entryTime(p).UTC().Format(time.RFC3339Nano), Author: &feeds.AtomAuthor{Name: author}, Content: &feeds.AtomContent{Type: "text", Content: text(first(props, "content"))}, Links: []feeds.AtomLink{{Href: p.Source, Rel: "alternate"}}}
		if t, err := parseDate(text(first(props, "published"))); err == nil {
			e.Published = t.UTC().Format(time.RFC3339Nano)
		}
		f.Entries = append(f.Entries, e)
	}
	b, e := xml.MarshalIndent(f, "", "  ")
	return append([]byte(xml.Header), b...), e
}
