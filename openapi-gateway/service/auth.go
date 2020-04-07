package service

import (
	"fmt"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
)

func parseAuth(auth string) (token, sub string, err error) {
	if strings.HasPrefix(auth, "Bearer ") {
		rawToken := auth[7:]
		sub, err = parseSubFromJwtToken(rawToken)
		if err != nil {
			err = fmt.Errorf("cannot parse bearer: %w", err)
			return
		}
		token = rawToken
		return
	}
	return "", "", fmt.Errorf("cannot parse bearer: prefix 'Bearer ' not found")
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
		return "", fmt.Errorf("cannot get sub from jwt token: %w", err)
	}

	if claims.Subject != "" {
		return claims.Subject, nil
	}

	return "", fmt.Errorf("cannot get sub from jwt token: not found")
}
