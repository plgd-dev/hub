package openid

import "fmt"

type Config struct {
	Issuer             string   `json:"issuer"`
	AuthURL            string   `json:"authorization_endpoint,omitempty"`
	TokenURL           string   `json:"token_endpoint"`
	JWKSURL            string   `json:"jwks_uri"`
	UserInfoURL        string   `json:"userinfo_endpoint,omitempty"`
	Algorithms         []string `json:"id_token_signing_alg_values_supported,omitempty"`
	EndSessionEndpoint string   `json:"end_session_endpoint,omitempty"`
	PlgdTokensEndpoint string   `json:"plgd_tokens_endpoint,omitempty"`
}

func (c Config) Validate() error {
	if c.JWKSURL == "" {
		return fmt.Errorf("jwks_uri('%v')", c.JWKSURL)
	}
	if c.TokenURL == "" {
		return fmt.Errorf("token_endpoint('%v')", c.TokenURL)
	}
	if c.Issuer == "" {
		return fmt.Errorf("issuer('%v')", c.Issuer)
	}
	return nil
}
