package http

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OnUnauthorizedAccessFunc = func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error)

// NewDefaultAuthorizationRules returns a map of HTTP methods to a slice of AuthArgs.
// The AuthArgs contain a URI field that is a regular expression matching the given apiPath
// with any path suffix. This function is used to create default authorization rules for
// HTTP methods GET, POST, DELETE, and PUT.
func NewDefaultAuthorizationRules(apiPath string) map[string][]pkgHttpJwt.AuthArgs {
	return map[string][]pkgHttpJwt.AuthArgs{
		http.MethodGet: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(apiPath) + AnyPathSuffixRegex),
			},
		},
		http.MethodPost: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(apiPath) + AnyPathSuffixRegex),
			},
		},
		http.MethodDelete: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(apiPath) + AnyPathSuffixRegex),
			},
		},
		http.MethodPut: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(apiPath) + AnyPathSuffixRegex),
			},
		},
	}
}

const (
	AnyPathSuffixRegex = `\/.*`
)

func GetToken(auth string) (string, error) {
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return auth[7:], nil
	}
	return "", status.Errorf(codes.Unauthenticated, "cannot parse bearer: prefix 'Bearer ' not found")
}

// CreateAuthMiddleware creates middleware for authorization
func CreateAuthMiddleware(authInterceptor pkgHttpJwt.Interceptor, onUnauthorizedAccessFunc OnUnauthorizedAccessFunc) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/": // health check
				next.ServeHTTP(w, r)
			default:
				token := r.Header.Get("Authorization")
				ctx := pkgHttpJwt.CtxWithToken(r.Context(), token)
				_, err := authInterceptor(ctx, r.Method, r.RequestURI)
				if err != nil {
					onUnauthorizedAccessFunc(ctx, w, r, err)
					return
				}

				if rawToken, err := GetToken(token); err == nil {
					r = r.WithContext(grpc.CtxWithToken(r.Context(), rawToken))
				}
				next.ServeHTTP(w, r)
			}
		})
	}
}
