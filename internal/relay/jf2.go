package relay

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

const jf2AuthorFallback = "投稿者不明"

// JF2は保存済みの並びをAtomと同じ順序・上限でJF2 Feedへ射影する。
func JF2(public, id string, posts []Post, limit int) ([]byte, error) {
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
	children := []any{}
	for _, p := range posts {
		children = append(children, jf2Entry(p))
	}
	f := map[string]any{
		"type":     "feed",
		"name":     "urn:uuid:" + id,
		"url":      strings.TrimRight(public, "/") + "/collections/" + id + ".jf2",
		"author":   map[string]any{"type": "card", "name": "indie-mezzanine"},
		"children": children,
	}
	return json.MarshalIndent(f, "", "  ")
}

// jf2Entryは保存済みh-entryを1件のentryへ射影する。
func jf2Entry(p Post) map[string]any {
	var raw map[string]any
	if e := json.Unmarshal([]byte(p.MF2), &raw); e != nil {
		raw = map[string]any{}
	}
	props, _ := raw["properties"].(map[string]any)
	out := jf2Properties(props)
	// 識別子は受付時のsource URLで固定し、保存済みmf2の値は使わない。
	delete(out, "uid")
	delete(out, "url")
	out["type"] = "entry"
	out["uid"] = p.Source
	out["url"] = p.Source
	// updatedはmf2の値、欠落時は初回保存日時とする。
	out["updated"] = entryTime(p).UTC().Format(time.RFC3339Nano)
	if name, _ := out["name"].(string); name == "" {
		out["name"] = p.Source
	}
	out["author"] = jf2Author(first(props, "author"))
	if children := jf2Children(raw["children"]); len(children) > 0 {
		out["children"] = children
	}
	return out
}

// jf2Propertiesは保存済みmf2のpropertiesをjf2の直接キーへ正規化する。
// 連絡先(email)と入れ子のvalue/id/lang/htmlは出力しない。
func jf2Properties(props map[string]any) map[string]any {
	out := map[string]any{}
	if _, ok := props["content"]; ok {
		if v, ok := jf2Content(first(props, "content")); ok {
			out["content"] = v
		}
	}
	if cats := jf2Categories(props["category"]); len(cats) > 0 {
		out["category"] = cats
	}
	if photo := text(first(props, "photo")); photo != "" {
		out["photo"] = photo
	}
	for _, k := range []string{"video", "audio"} {
		if v, ok := jf2Media(props[k]); ok {
			out[k] = v
		}
	}
	for _, k := range []string{"published", "updated"} {
		if t, e := parseDate(text(first(props, k))); e == nil {
			out[k] = t.UTC().Format(time.RFC3339Nano)
		}
	}
	for k, raw := range props {
		switch k {
		case "email", "content", "category", "photo", "video", "audio", "published", "updated":
			continue
		}
		if v, ok := jf2Value(raw); ok {
			out[k] = v
		}
	}
	return out
}

// jf2Objectは入れ子の型付きオブジェクトを直接キー表現へ正規化する。
func jf2Object(m map[string]any) map[string]any {
	out := map[string]any{}
	if t := jf2Type(m["type"]); t != "" {
		out["type"] = t
	}
	if props, ok := m["properties"].(map[string]any); ok {
		for k, v := range jf2Properties(props) {
			out[k] = v
		}
	}
	if children := jf2Children(m["children"]); len(children) > 0 {
		out["children"] = children
	}
	return out
}

// jf2Authorはentryのauthorを名前付きのcardへ固定する。
func jf2Author(v any) map[string]any {
	card, _ := v.(map[string]any)
	if card == nil {
		return map[string]any{"type": "card", "name": jf2AuthorFallback}
	}
	out := jf2Object(card)
	if _, ok := out["type"]; !ok {
		out["type"] = "card"
	}
	if name, _ := out["name"].(string); name == "" {
		out["name"] = jf2AuthorFallback
	}
	return out
}

// jf2Contentは本文をtextだけのオブジェクトへ固定する。
func jf2Content(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	return map[string]any{"text": text(v)}, true
}

// jf2Categoriesはcategoryを文字列の配列へ固定する。
func jf2Categories(raw any) []any {
	vals, _ := raw.([]any)
	out := []any{}
	for _, v := range vals {
		if s := text(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// jf2Mediaはvideo/audioをurlを持つオブジェクトへ固定する。
func jf2Media(raw any) (any, bool) {
	vals, _ := raw.([]any)
	out := []any{}
	for _, v := range vals {
		if s := text(v); s != "" {
			out = append(out, map[string]any{"url": s})
		}
	}
	return jf2Condense(out)
}

// jf2Valueは型付きオブジェクトとvalue付きオブジェクトを解決し、単一要素の配列を単一値へ畳む。
func jf2Value(raw any) (any, bool) {
	vals, ok := raw.([]any)
	if !ok {
		vals = []any{raw}
	}
	out := []any{}
	for _, v := range vals {
		switch x := v.(type) {
		case string:
			out = append(out, x)
		case map[string]any:
			if jf2Type(x["type"]) != "" {
				out = append(out, jf2Object(x))
			} else if s, ok := x["value"].(string); ok {
				out = append(out, s)
			}
		}
	}
	return jf2Condense(out)
}

func jf2Condense(vals []any) (any, bool) {
	switch len(vals) {
	case 0:
		return nil, false
	case 1:
		return vals[0], true
	default:
		return vals, true
	}
}

func jf2Children(raw any) []any {
	vals, _ := raw.([]any)
	out := []any{}
	for _, v := range vals {
		if m, ok := v.(map[string]any); ok {
			out = append(out, jf2Object(m))
		}
	}
	return out
}

// jf2Typeはtypeを単一文字列にし、h-接頭辞を除く。
func jf2Type(v any) string {
	types, _ := v.([]any)
	for _, t := range types {
		if s, ok := t.(string); ok {
			return strings.TrimPrefix(s, "h-")
		}
	}
	return ""
}
