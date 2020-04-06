package store

import (
	"context"
	"fmt"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"golang.org/x/oauth2"
)

type AccessToken string

func (t AccessToken) String() string {
	return string(t)
}

type OAuth struct {
	LinkedCloudID string
	AccessToken   AccessToken
	RefreshToken  string
	Expiry        time.Time
}

type LinkedCloudsHandler struct {
	LinkedClouds []LinkedCloud
}

func (h *LinkedCloudsHandler) Handle(ctx context.Context, iter LinkedCloudIter) (err error) {
	var s LinkedCloud
	for iter.Next(ctx, &s) {
		h.LinkedClouds = append(h.LinkedClouds, s)
	}
	return iter.Err()
}

func (o OAuth) Refresh(ctx context.Context, s Store) (OAuth, error) {
	if o.Expiry.IsZero() {
		return o, nil
	}
	var h LinkedCloudsHandler
	err := s.LoadLinkedClouds(ctx, Query{ID: o.LinkedCloudID}, &h)
	if err != nil {
		return o, err
	}
	if len(h.LinkedClouds) != 1 {
		return o, fmt.Errorf("linked cloud %v not found", o.LinkedCloudID)
	}
	l := h.LinkedClouds[0]
	c := l.ToOAuth2Config()
	restoredToken := oauth2.Token{
		RefreshToken: o.RefreshToken,
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, http.DefaultClient)
	tokenSource := c.TokenSource(ctx, &restoredToken)
	token, err := tokenSource.Token()
	if err != nil {
		return o, err
	}
	return OAuth{
		LinkedCloudID: o.LinkedCloudID,
		AccessToken:   AccessToken(token.AccessToken),
		Expiry:        token.Expiry,
		RefreshToken:  token.RefreshToken,
	}, nil
}

func (o OAuth) IsValidAccessToken() bool {
	if o.Expiry.IsZero() || o.Expiry.UnixNano() > time.Now().UnixNano() {
		return true
	}
	return false
}

func (o OAuth) GetAccessToken() (AccessToken, error) {
	if o.IsValidAccessToken() {
		return o.AccessToken, nil
	}
	return AccessToken(""), fmt.Errorf("cannot get accesstoken: token is invalid")
}

type claims struct {
	Subject string `json:"sub,omitempty"`
}

func (c *claims) Valid() error {
	return nil
}

func parseSubFromJwtToken(rawJwtToken string) (string, error) {
	parser := &jwt.Parser{
		SkipClaimsValidation: true,
	}

	var claims claims
	_, _, err := parser.ParseUnverified(rawJwtToken, &claims)
	if err != nil {
		return "", fmt.Errorf("cannot get subject from jwt token: %v", err)
	}

	if claims.Subject != "" {
		return claims.Subject, nil
	}

	return "", fmt.Errorf("cannot get subject from jwt token: not found")
}

func (t AccessToken) GetSubject() (string, error) {
	return parseSubFromJwtToken(string(t))
}

type LinkedAccount struct {
	ID          string
	TargetURL   string
	TargetCloud OAuth
	OriginCloud OAuth
}

func (l LinkedAccount) RefreshTokens(ctx context.Context, s Store) (LinkedAccount, error) {
	if l.TargetCloud.IsValidAccessToken() && l.OriginCloud.IsValidAccessToken() {
		return l, nil
	}
	t := l.TargetCloud
	o := l.OriginCloud
	var err error
	if !t.IsValidAccessToken() {
		t, err = t.Refresh(ctx, s)
		if err != nil {
			return l, fmt.Errorf("cannot refreash target cloud access token: %v", err)
		}
	}
	if !o.IsValidAccessToken() {
		o, err = o.Refresh(ctx, s)
		if err != nil {
			return l, fmt.Errorf("cannot refresh target cloud access token: %v", err)
		}
	}
	l.TargetCloud = t
	l.OriginCloud = o

	err = s.UpdateLinkedAccount(ctx, l)
	if err != nil {
		return l, fmt.Errorf("cannot store updated linked account: %v", err)
	}
	return l, nil
}
