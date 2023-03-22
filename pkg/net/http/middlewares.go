package http

import (
	"net/http"
	"net/url"
	"strings"
)

func CreateMakeQueryCaseInsensitiveMiddleware(queryCaseInsensitive map[string]string, opts ...LogOpt) func(next http.Handler) http.Handler {
	cfg := NewLogOptions()
	for _, o := range opts {
		o(cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, err := url.ParseRequestURI(r.RequestURI)
			if err != nil {
				cfg.logger.Errorf("cannot make query case insensitive: %w", err)
				next.ServeHTTP(w, r)
				return
			}
			queries := u.Query()
			newQueries := make(url.Values)
			for key, val := range queries {
				newKey, ok := queryCaseInsensitive[strings.ToLower(key)]
				if ok {
					newQueries[newKey] = val
				} else {
					newQueries[key] = val
				}
			}
			r.URL.RawQuery = newQueries.Encode()
			r.RequestURI = u.String()
			next.ServeHTTP(w, r)
		})
	}
}

func CreateTrailSlashSuffixMiddleware(opts ...LogOpt) func(next http.Handler) http.Handler {
	cfg := NewLogOptions()
	for _, o := range opts {
		o(cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, err := url.ParseRequestURI(r.RequestURI)
			if err != nil {
				cfg.logger.Errorf("cannot trail slash suffix: %v", err)
				next.ServeHTTP(w, r)
				return
			}
			l := len(u.Path)
			u.Path = strings.TrimRight(u.Path, "/")
			if l > 0 && len(u.Path) == 0 {
				u.Path = "/"
			}
			r.RequestURI = u.String()
			r.URL.Path = u.Path
			next.ServeHTTP(w, r)
		})
	}
}
