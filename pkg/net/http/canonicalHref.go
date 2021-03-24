package http

import (
	"regexp"
	"strings"
)

// CanonicalHref always lead by "/"
func CanonicalHref(href string) string {
	backslash := regexp.MustCompile(`\/+`)
	p := backslash.ReplaceAllString(href, "/")
	p = strings.TrimLeft(p, "/")
	p = strings.TrimRight(p, "/")

	return "/" + p
}
