package relay

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	mdtext "github.com/yuin/goldmark/text"
)

//go:embed landing.md
var landingMarkdown []byte

const landingTitle = "indie-mezzanine"

// landingDocument renders the embedded Markdown source into the root page
// document, resolving root-relative destinations against publicURL. It is
// called once at startup so requests only write the cached result.
func landingDocument(publicURL string) ([]byte, error) {
	base, e := url.Parse(publicURL)
	if e != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("公開URLを基底にできない: %q", publicURL)
	}
	entry, e := renderEntry(landingMarkdown, base)
	if e != nil {
		return nil, e
	}
	endpoint := strings.TrimRight(publicURL, "/") + "/webmention"
	var doc bytes.Buffer
	fmt.Fprintf(&doc, `<!doctype html><html lang="ja"><head><meta charset="utf-8"><title>%s</title><link rel="webmention" href="%s"></head><body>`, landingTitle, html.EscapeString(endpoint))
	doc.Write(entry)
	doc.WriteString("</body></html>")
	return doc.Bytes(), nil
}

// renderEntry renders source as a single MF2 h-entry: the level-1 heading
// names the entry, and the remaining blocks form its content.
func renderEntry(source []byte, base *url.URL) ([]byte, error) {
	md, doc, e := parseLanding(source, base)
	if e != nil {
		return nil, e
	}
	name, rest := splitEntry(doc)
	if name == nil {
		return nil, fmt.Errorf("主見出しがない")
	}
	name.SetAttributeString("class", "p-name")
	var buf bytes.Buffer
	buf.WriteString(`<article class="h-entry">`)
	if e := md.Renderer().Render(&buf, source, name); e != nil {
		return nil, e
	}
	buf.WriteString(`<div class="e-content">`)
	for _, n := range rest {
		if e := md.Renderer().Render(&buf, source, n); e != nil {
			return nil, e
		}
	}
	buf.WriteString(`</div></article>`)
	return buf.Bytes(), nil
}

// splitEntry separates the level-1 heading that names the entry from the
// blocks that form its content.
func splitEntry(doc ast.Node) (ast.Node, []ast.Node) {
	var name ast.Node
	var rest []ast.Node
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		if name == nil {
			if h, ok := c.(*ast.Heading); ok && h.Level == 1 {
				name = c
				continue
			}
		}
		rest = append(rest, c)
	}
	return name, rest
}

// parseLanding parses source and rewrites root-relative destinations under
// base, returning the renderer and the resolved document.
func parseLanding(source []byte, base *url.URL) (goldmark.Markdown, ast.Node, error) {
	md := goldmark.New()
	doc := md.Parser().Parse(mdtext.NewReader(source))
	if e := absolutize(doc, base); e != nil {
		return nil, nil, e
	}
	return md, doc, nil
}

// renderMarkdown parses source and renders HTML, rewriting root-relative
// link and image destinations to absolute URLs under base.
func renderMarkdown(source []byte, base *url.URL) ([]byte, error) {
	md, doc, e := parseLanding(source, base)
	if e != nil {
		return nil, e
	}
	var buf bytes.Buffer
	if e := md.Renderer().Render(&buf, source, doc); e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}

func absolutize(doc ast.Node, base *url.URL) error {
	prefix := strings.TrimRight(base.String(), "/")
	return ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Link:
			v.Destination = resolveDestination(v.Destination, prefix)
		case *ast.Image:
			v.Destination = resolveDestination(v.Destination, prefix)
		}
		return ast.WalkContinue, nil
	})
}

// resolveDestination turns an application root path into an absolute URL.
// Protocol-relative, external, mailto and fragment destinations are kept.
func resolveDestination(destination []byte, prefix string) []byte {
	d := string(destination)
	if strings.HasPrefix(d, "/") && !strings.HasPrefix(d, "//") {
		return []byte(prefix + d)
	}
	return destination
}
