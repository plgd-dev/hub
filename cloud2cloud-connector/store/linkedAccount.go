package store

import (
	"context"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"golang.org/x/oauth2"
)

type AccessToken string

func (t AccessToken) String() string {
	return string(t)
}

type Token struct {
	AccessToken  AccessToken
	RefreshToken string
	Expiry       time.Time
}

type LinkedCloudsHandler struct {
	LinkedClouds []LinkedCloud
}

func (h *LinkedCloudsHandler) Handle(ctx context.Context, iter LinkedCloudIter) (err error) {
	for {
		var s LinkedCloud
		if !iter.Next(ctx, &s) {
			break
		}
		h.LinkedClouds = append(h.LinkedClouds, s)
	}
	return iter.Err()
}

func (o Token) Refresh(ctx context.Context, cfg oauth2.Config) (Token, bool, error) {
	if o.IsValidAccessToken() {
		return o, false, nil
	}
	if o.Expiry.IsZero() {
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

func (o Token) GetAccessToken() (AccessToken, error) {
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
		return "", fmt.Errorf("cannot get subject from jwt token: %w", err)
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
	ID            string `json:"Id" bson:"_id"`
	LinkedCloudID string `bson:"linkedcloudid"`
	UserID        string
	TargetCloud   Token
}

/*
func (l LinkedAccount) RefreshToken(ctx context.Context, cfg oauth2.Config) (LinkedAccount, bool, error) {
	if l.TargetCloud.IsValidAccessToken() {
		return l, nil
	}
	t := l.TargetCloud
	var err error
	if !t.IsValidAccessToken() {
		t, err = t.Refresh(ctx, cfg)
		if err != nil {
			return l, fmt.Errorf("cannot refreash target cloud access token: %w", err)
		}
	}
	l.TargetCloud = t
	err = s.UpdateLinkedAccount(ctx, l)
	if err != nil {
		return l, fmt.Errorf("cannot store updated linked account: %w", err)
	}
	return l, nil
}
*/
