package service

import (
	"time"

	pkgTime "github.com/plgd-dev/cloud/v2/pkg/time"
)

// ValidUntil returns time until expiration.
// No expiration is denoted by expiry = 0 for which 0 is returned.
// When expired, ok is false.
func ValidUntil(expiry time.Time) (validUntil int64, ok bool) {
	if expiry.IsZero() {
		return 0, true // No expiration.
	}
	if time.Now().Before(expiry) {
		return pkgTime.UnixNano(expiry), true
	}
	return pkgTime.UnixNano(expiry), false
}
