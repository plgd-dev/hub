package uri

import (
	"regexp"
	"strings"
)

// CanonicalHref always lead by "/"
func CanonicalHref(href string) string {
	p := CanonicalURI(href)
	p = strings.TrimLeft(p, "/")
	return "/" + p
}

func CanonicalURI(uri string) string {
	var schema string
	href := uri
	components := strings.SplitN(uri, "://", 2)
	if len(components) > 1 {
		schema = components[0] + "://"
		href = components[1]
	}

	backslash := regexp.MustCompile(`\/+`)
	p := backslash.ReplaceAllString(href, "/")
	p = strings.TrimRight(p, "/")
	return schema + p
}
