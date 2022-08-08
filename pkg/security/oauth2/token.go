package oauth2

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

type AccessToken string

func (t AccessToken) String() string {
	return string(t)
}

// Token provides access tokens and their attributes.
type Token struct {
	AccessToken  AccessToken
	RefreshToken string
	Expiry       time.Time
}

func (o Token) Refresh(ctx context.Context, cfg oauth2.Config) (Token, bool, error) {
	if o.IsValidAccessToken() {
		return o, false, nil
	}
	restoredToken := oauth2.Token{
		RefreshToken: o.RefreshToken,
	}
	tokenSource := cfg.TokenSource(ctx, &restoredToken)
	token, err := tokenSource.Token()
	if err != nil {
		return o, false, err
	}
	return Token{
		AccessToken:  AccessToken(token.AccessToken),
		Expiry:       token.Expiry,
		RefreshToken: token.RefreshToken,
	}, true, nil
}

func (o Token) IsValidAccessToken() bool {
	if o.Expiry.IsZero() || o.Expiry.UnixNano() > time.Now().UnixNano() {
		return true
	}
	return false
}
