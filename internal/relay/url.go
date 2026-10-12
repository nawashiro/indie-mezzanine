package relay

import "net/url"

// rootURLString changes only an HTTP(S) root's empty path to /.
// Callers retain the original source string as the snapshot identity.
func rootURLString(u *url.URL) string {
	v := *u
	if (v.Scheme == "http" || v.Scheme == "https") && v.Host != "" && v.Opaque == "" && v.Path == "" {
		v.Path = "/"
	}
	return v.String()
}
