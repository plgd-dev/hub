package provider

import (
	"time"
)

// Token provides access tokens and their attributes.
type Token struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	Owner        string
}
