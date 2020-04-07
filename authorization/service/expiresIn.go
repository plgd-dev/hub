package service

import "time"

// ExpiresIn calculates the remaining time until expiration.
// No expiration is denoted by expiry = 0 for which -1 is returned.
// When expired, ok is false.
func ExpiresIn(expiry time.Time) (expiresInSeconds int64, ok bool) {
	if expiry.IsZero() {
		return -1, true // No expiration.
	}
	d := int64(expiry.Sub(time.Now()).Seconds())
	if d <= 0 {
		return d, false
	}
	return d, true
}
