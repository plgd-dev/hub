package service

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

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
