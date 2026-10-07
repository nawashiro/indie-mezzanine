package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/net/html"
	"net/url"
	"regexp"
	"strings"
	"willnorris.com/go/microformats"
)

var ErrIneligible = errors.New("投稿資格なし")
var urnPattern = regexp.MustCompile(`^urn:uuid:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func Collection(raw string) (string, error) {
	if !urnPattern.MatchString(raw) {
		return "", ErrIneligible
	}
	id, e := uuid.Parse(raw[9:])
	if e != nil {
		return "", ErrIneligible
	}
	return id.String(), nil
}

type Record struct {
	Collection string
	MF2        json.RawMessage
}

func walk(n *html.Node, f func(*html.Node)) {
	f(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, f)
	}
}
func attr(n *html.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}
func Parse(p Page, target string) (Record, error) {
	base, e := url.Parse(p.URL)
	if e != nil {
		return Record{}, e
	}
	doc, e := html.Parse(bytes.NewReader(p.Body))
	if e != nil {
		return Record{}, e
	}
	baseFound := false
	walk(doc, func(n *html.Node) {
		if !baseFound && n.Type == html.ElementNode && n.Data == "base" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					baseFound = true
					if u, e := url.Parse(a.Val); e == nil {
						base = base.ResolveReference(u)
					}
				}
			}
		}
	})
	linked := false
	cols := map[string]bool{}
	invalid := false
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if n.Data == "a" || n.Data == "link" {
			for _, rel := range strings.Fields(attr(n, "rel")) {
				if rel == "collection" {
					id, e := Collection(attr(n, "href"))
					if e != nil {
						invalid = true
					} else {
						cols[id] = true
					}
				}
			}
		}
		key := ""
		switch n.Data {
		case "a", "area", "link", "img":
			key = "href"
		case "audio", "video", "source", "track", "iframe", "embed", "script", "input":
			key = "src"
		case "object":
			key = "data"
		}
		for _, a := range n.Attr {
			if a.Key == key || (n.Data == "img" && a.Key == "src") || (n.Data == "video" && a.Key == "poster") {
				u, e := url.Parse(a.Val)
				if e == nil && base.ResolveReference(u).String() == target {
					linked = true
				}
			}
		}
	})
	if !linked || invalid || len(cols) != 1 {
		return Record{}, ErrIneligible
	}
	data := microformats.ParseNode(doc, base)
	var entries []*microformats.Microformat
	var scan func(*microformats.Microformat)
	scan = func(m *microformats.Microformat) {
		for _, t := range m.Type {
			if t == "h-entry" {
				entries = append(entries, m)
				break
			}
		}
		for _, ch := range m.Children {
			scan(ch)
		}
		for _, values := range m.Properties {
			for _, v := range values {
				if x, ok := v.(*microformats.Microformat); ok {
					scan(x)
				}
			}
		}
	}
	for _, m := range data.Items {
		scan(m)
	}
	if len(entries) != 1 {
		return Record{}, ErrIneligible
	}
	raw, e := json.Marshal(entries[0])
	if e != nil {
		return Record{}, e
	}
	var clean any
	if e = json.Unmarshal(raw, &clean); e != nil {
		return Record{}, e
	}
	stripHTML(clean)
	raw, e = json.Marshal(clean)
	if e != nil {
		return Record{}, e
	}
	var id string
	for k := range cols {
		id = k
	}
	return Record{id, raw}, nil
}
func stripHTML(v any) {
	switch x := v.(type) {
	case map[string]any:
		delete(x, "html")
		for _, v := range x {
			stripHTML(v)
		}
	case []any:
		for _, v := range x {
			stripHTML(v)
		}
	}
}
