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
	body, e := renderMarkdown(landingMarkdown, base)
	if e != nil {
		return nil, e
	}
	endpoint := strings.TrimRight(publicURL, "/") + "/webmention"
	var doc bytes.Buffer
	fmt.Fprintf(&doc, `<!doctype html><html lang="ja"><head><meta charset="utf-8"><title>%s</title><link rel="webmention" href="%s"></head><body>`, landingTitle, html.EscapeString(endpoint))
	doc.Write(body)
	doc.WriteString("</body></html>")
	return doc.Bytes(), nil
}

// renderMarkdown parses source and renders HTML, rewriting root-relative
// link and image destinations to absolute URLs under base.
func renderMarkdown(source []byte, base *url.URL) ([]byte, error) {
	md := goldmark.New()
	doc := md.Parser().Parse(mdtext.NewReader(source))
	if e := absolutize(doc, base); e != nil {
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
