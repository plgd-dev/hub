package validator_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	pkgTls "github.com/plgd-dev/hub/v2/pkg/security/tls"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		args    validator.Config
		wantErr bool
	}{
		{
			name: "valid",
			args: validator.Config{
				Audience: "example-audience",
				Endpoints: []validator.AuthorityConfig{
					{
						Authority: "example-address",
						HTTP:      config.MakeHttpClientConfig(),
					},
				},
			},
		},
		{
			name: "empty endpoints",
			args: validator.Config{
				Audience: "example-audience",
			},
			wantErr: true,
		},
		{
			name: "empty address",
			args: validator.Config{
				Audience: "example-audience",
				Endpoints: []validator.AuthorityConfig{
					{
						HTTP: config.MakeHttpClientConfig(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "deprecated fields",
			args: validator.Config{
				Audience: "example-audience",
				Authority: func() *string {
					s := "example-authority"
					return &s
				}(),
				HTTP: func() *pkgTls.HTTPConfig {
					c := config.MakeHttpClientConfig()
					return &c
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
