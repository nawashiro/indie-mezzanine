package relay

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	PublicURL, DataDir, Listen                     string
	FetchTimeout                                   time.Duration
	MaxBody, MaxForm                               int64
	Redirects, Workers, Queue, FeedLimit, Attempts int
	Retention                                      time.Duration
}

func DefaultConfig() Config {
	return Config{Listen: ":8080", FetchTimeout: 10 * time.Second, MaxBody: 1 << 20, MaxForm: 16 << 10, Redirects: 5, Workers: 2, Queue: 1000, FeedLimit: 100, Attempts: 3, Retention: 7 * 24 * time.Hour}
}
func (c Config) Validate() error {
	u, e := url.Parse(c.PublicURL)
	if e != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("PUBLIC_URL は認証情報・query・fragmentなしのHTTP(S)ルートURL（末尾 /）が必要")
	}
	if c.DataDir == "" || c.Listen == "" || c.FetchTimeout <= 0 || c.MaxBody <= 0 || c.MaxForm <= 0 || c.Redirects <= 0 || c.Workers <= 0 || c.Queue <= 0 || c.FeedLimit <= 0 || c.Attempts <= 0 || c.Retention <= 0 {
		return fmt.Errorf("データディレクトリと有限の正の資源上限が必要")
	}
	return nil
}
func ConfigFromEnv() (Config, error) {
	c := DefaultConfig()
	c.PublicURL = os.Getenv("PUBLIC_URL")
	c.DataDir = os.Getenv("DATA_DIR")
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		c.Listen = v
	}
	ints := map[string]*int{"WORKERS": &c.Workers, "QUEUE_LIMIT": &c.Queue, "FEED_LIMIT": &c.FeedLimit, "MAX_REDIRECTS": &c.Redirects, "MAX_ATTEMPTS": &c.Attempts}
	for k, p := range ints {
		if v := os.Getenv(k); v != "" {
			n, e := strconv.Atoi(v)
			if e != nil {
				return c, fmt.Errorf("%s: %w", k, e)
			}
			*p = n
		}
	}
	for k, p := range map[string]*int64{"MAX_BODY_BYTES": &c.MaxBody, "MAX_FORM_BYTES": &c.MaxForm} {
		if v := os.Getenv(k); v != "" {
			n, e := strconv.ParseInt(v, 10, 64)
			if e != nil {
				return c, fmt.Errorf("%s: %w", k, e)
			}
			*p = n
		}
	}
	for k, p := range map[string]*time.Duration{"FETCH_TIMEOUT": &c.FetchTimeout, "JOB_RETENTION": &c.Retention} {
		if v := os.Getenv(k); v != "" {
			n, e := time.ParseDuration(v)
			if e != nil {
				return c, fmt.Errorf("%s: %w", k, e)
			}
			*p = n
		}
	}
	return c, c.Validate()
}
