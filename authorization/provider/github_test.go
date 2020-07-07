package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"io"

	"github.com/go-ocf/cloud/authorization/oauth"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
)

const providerName = "github"

func TestAuthCodeURL(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	p := newGithubOAuth(server)

	token := "randomToken"
	url := p.AuthCodeURL(token)
	assert.Contains(t, url, token)
}

func TestSignUpGitHubProvider(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/access_token", makeJSONHandler(http.StatusOK, validAccessToken))
	mux.HandleFunc("/github/api/", makeJSONHandler(http.StatusOK, validUserID))
	server := httptest.NewServer(mux)
	defer server.Close()

	p := newGithubOAuth(server)
	ctx := context.Background()
	token, err := p.Exchange(ctx, providerName, "authCode")

	assert := assert.New(t)
	assert.Nil(err)
	assert.Equal("TestAccessToken", token.AccessToken)
	assert.Equal("TestRefreshToken", token.RefreshToken)
	expiresIn := int(token.Expiry.Sub(time.Now()).Seconds())
	assert.True(3595 < expiresIn && expiresIn <= 3600)
	assert.Equal("42", token.UserID)
}

func TestRefreshTokenGitHubProvider(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/access_token", makeJSONHandler(http.StatusOK, validAccessToken))
	mux.HandleFunc("/github/api/", makeJSONHandler(http.StatusOK, validUserID))
	server := httptest.NewServer(mux)
	defer server.Close()

	p := newGithubOAuth(server)
	ctx := context.Background()
	n, err := p.Refresh(ctx, "refresh-token")

	assert := assert.New(t)
	assert.Nil(n)
	assert.Error(err)
}

func TestOAuthExchangeFailure(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	p := newGithubOAuth(server)
	ctx := context.Background()
	token, err := p.Exchange(ctx, providerName, "authCode")

	assert := assert.New(t)
	assert.Error(err)
	assert.Nil(token)
}

func TestGithubFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/access_token", makeJSONHandler(http.StatusOK, validAccessToken))
	server := httptest.NewServer(mux)
	defer server.Close()

	p := newGithubOAuth(server)
	ctx := context.Background()
	token, err := p.Exchange(ctx, providerName, "authCode")

	assert := assert.New(t)
	assert.Error(err)
	assert.Nil(token)
}

func newGithubOAuth(server *httptest.Server) Provider {
	endpoint := oauth.Endpoint{
		AuthURL:  server.URL + "/oauth/authorize",
		TokenURL: server.URL + "/oauth/access_token",
	}
	Config := Config{
		Provider: "github",
		OAuth2: oauth.Config{
			ClientID:     "clientId",
			ClientSecret: "clientSecret",
			RedirectURL:  "",
			Endpoint:     endpoint,
		},
	}
	p := NewGitHubProvider(Config)
	p.NewGithubClient = func(h *http.Client) *github.Client {
		c := github.NewClient(h)
		baseURL, _ := url.Parse(server.URL + "/github/api/")
		c.BaseURL = baseURL
		return c
	}
	p.NewHTTPClient = server.Client

	return p
}

func makeJSONHandler(statuscode int, body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statuscode)
		io.WriteString(w, body)
	}
}

// https://www.oauth.com/oauth2-servers/access-tokens/access-token-response/
const validAccessToken = `{
  "access_token":"TestAccessToken",
  "token_type":"TestTokenType",
  "expires_in": 3600,
  "refresh_token":"TestRefreshToken",
  "scope":"TestScope"
}
`

// https://developer.github.com/v3/users/
const validUserID = `{ 
	"id": 42 
}
`
