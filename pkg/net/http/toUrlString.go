package http

import "net/url"

// ToURLString convert scheme, host, path to escaped url.
func ToURLString(scheme string, host string, path string) string {
	url := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}
	return url.String()
}
