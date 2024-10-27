package x509

import (
	"fmt"
	"net/url"
)

func ValidateCRLDistributionPointAddress(s string) error {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return fmt.Errorf("invalid address(%s)", s)
	}
	if !u.IsAbs() {
		return fmt.Errorf("invalid relative URL address(%s)", s)
	}
	return nil
}
