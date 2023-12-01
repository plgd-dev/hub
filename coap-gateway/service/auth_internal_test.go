//go:build test
// +build test

package service

import (
	"testing"

	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/stretchr/testify/require"
)

func TestVerifyAndResolveDeviceID(t *testing.T) {
	makeConfig := func(deviceIDClaim string, tlsEnabled, certificateRequired bool) Config {
		var cfg Config
		cfg.APIs.COAP.Authorization.DeviceIDClaim = deviceIDClaim
		if tlsEnabled {
			cfg.APIs.COAP.TLS.Enabled = new(bool)
			*cfg.APIs.COAP.TLS.Enabled = true
		}
		cfg.APIs.COAP.TLS.Embedded.ClientCertificateRequired = certificateRequired
		return cfg
	}

	type args struct {
		cfg           Config
		tlsDeviceID   string
		paramDeviceID string
		claim         pkgJwt.Claims
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    string
	}{
		{
			name: "invalid device id claim type",
			args: args{
				cfg:   makeConfig("sub", false, false),
				claim: pkgJwt.Claims{"sub": 42},
			},
			wantErr: true,
		},
		{
			name: "invalid device id claim",
			args: args{
				cfg:   makeConfig("sub", false, false),
				claim: pkgJwt.Claims{"sub": ""},
			},
			wantErr: true,
		},
		{
			name: "invalid TLS DeviceID",
			args: args{
				cfg:   makeConfig("sub", true, true),
				claim: pkgJwt.Claims{"sub": "user"},
			},
			wantErr: true,
		},
		{
			name: "non-matching device ids",
			args: args{
				cfg:         makeConfig("sub", true, true),
				tlsDeviceID: "tlsUser",
				claim:       pkgJwt.Claims{"sub": "user"},
			},
			wantErr: true,
		},
		{
			name: "valid - device id from claim",
			args: args{
				cfg:   makeConfig("sub", false, false),
				claim: pkgJwt.Claims{"sub": "user"},
			},
			want: "user",
		},
		{
			name: "valid - tls device id",
			args: args{
				cfg:         makeConfig("sub", true, true),
				tlsDeviceID: "user",
				claim:       pkgJwt.Claims{"sub": "user"},
			},
			want: "user",
		},
		{
			name: "valid - tls device id, deviceID claim",
			args: args{
				cfg:         makeConfig("", true, true),
				tlsDeviceID: "tlsUser",
				claim:       pkgJwt.Claims{},
			},
			want: "tlsUser",
		},
		{
			name: "valid - param deviceID",
			args: args{
				cfg:           makeConfig("", false, false),
				paramDeviceID: "paramUser",
				claim:         pkgJwt.Claims{"sub": "user"},
			},
			want: "paramUser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				config: tt.args.cfg,
			}
			got, err := s.VerifyAndResolveDeviceID(tt.args.tlsDeviceID, tt.args.paramDeviceID, tt.args.claim)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
