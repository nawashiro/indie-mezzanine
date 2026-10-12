package relay

import (
	"net/url"
	"testing"
)

func TestRootURLString(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		base := scheme + "://relay.example"
		for _, suffix := range []string{"", "/"} {
			u, _ := url.Parse(base + suffix)
			before := u.String()
			if got := rootURLString(u); got != base+"/" || u.String() != before {
				t.Fatalf("normalization mutated input or wrong result: %s", got)
			}
		}
	}
	for _, raw := range []string{"https://other.example", "http://relay.example", "https://RELAY.example", "https://relay.example:443", "https://relay.example/post", "https://relay.example/post/", "https://relay.example//", "https://relay.example/%2f", "https://relay.example?", "https://relay.example?x", "https://relay.example#x", "https://user@relay.example"} {
		u, _ := url.Parse(raw)
		if rootURLString(u) == "https://relay.example/" {
			t.Fatalf("unexpected equivalence: %s", raw)
		}
	}
}

func TestRootConfigCompatibility(t *testing.T) {
	c := config(t)
	for _, raw := range []string{"http://relay.example", "http://relay.example/", "https://relay.example", "https://relay.example/", "https://relay.example#", "https://relay.example/#"} {
		c.PublicURL = raw
		if e := c.Validate(); e != nil {
			t.Fatalf("previously accepted config %s: %v", raw, e)
		}
	}
	for _, raw := range []string{"https://relay.example?", "https://relay.example?x", "https://relay.example#x", "https://user@relay.example", "https://relay.example/post", "https://relay.example/%2f", "ftp://relay.example", "https:///"} {
		c.PublicURL = raw
		if c.Validate() == nil {
			t.Fatalf("invalid config accepted: %s", raw)
		}
	}
}
