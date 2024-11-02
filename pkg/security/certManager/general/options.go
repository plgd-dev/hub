package general

import pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"

type Options struct {
	CustomDistributionPointVerification pkgX509.CustomDistributionPointVerification
}

type SetOption = func(cfg *Options)

func WithCustomDistributionPointVerification(customDistributionPointVerification pkgX509.CustomDistributionPointVerification) SetOption {
	return func(o *Options) {
		o.CustomDistributionPointVerification = customDistributionPointVerification
	}
}
