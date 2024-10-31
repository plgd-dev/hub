package tls_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/security/tls"
	"github.com/stretchr/testify/require"
)

func TestClientConfigValidate(t *testing.T) {
	validClientConfig := func() *tls.ClientConfig {
		return &tls.ClientConfig{
			CAPool:          []string{"file://path/to/ca1.pem", "file://path/to/ca2.pem"},
			KeyFile:         "file://path/to/key.pem",
			CertFile:        "file://path/to/cert.pem",
			UseSystemCAPool: false,
		}
	}

	type args struct {
		cfg *tls.ClientConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				cfg: validClientConfig(),
			},
		},
		{
			name: "invalid ca pool type",
			args: args{
				cfg: func() *tls.ClientConfig {
					c := validClientConfig()
					c.CAPool = 42
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "empty ca pool",
			args: args{
				cfg: func() *tls.ClientConfig {
					c := validClientConfig()
					c.CAPool = []string{}
					c.UseSystemCAPool = false
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "missing keyfile",
			args: args{
				cfg: func() *tls.ClientConfig {
					c := validClientConfig()
					c.KeyFile = ""
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "missing certfile",
			args: args{
				cfg: func() *tls.ClientConfig {
					c := validClientConfig()
					c.CertFile = ""
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "crl invalid - missing http",
			args: args{
				cfg: func() *tls.ClientConfig {
					c := validClientConfig()
					c.CRL.Enabled = true
					return c
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClientConfigEquals(t *testing.T) {
	cfg1 := tls.ClientConfig{
		CAPool:          []string{"file://path/to/ca1.pem", "file://path/to/ca2.pem"},
		KeyFile:         "file://path/to/key.pem",
		CertFile:        "file://path/to/cert.pem",
		UseSystemCAPool: false,
	}

	type args struct {
		cfg1 tls.ClientConfig
		cfg2 tls.ClientConfig
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal configurations",
			args: args{
				cfg1: cfg1,
				cfg2: cfg1,
			},
			want: true,
		},
		{
			name: "different ca pool",
			args: args{
				cfg1: cfg1,
				cfg2: func() tls.ClientConfig {
					c := cfg1
					c.CAPool = []string{"file://path/to/ca2.pem"}
					return c
				}(),
			},
			want: false,
		},
		{
			name: "different keyfile",
			args: args{
				cfg1: cfg1,
				cfg2: func() tls.ClientConfig {
					c := cfg1
					c.KeyFile = "file://path/to/otherkey.pem"
					return c
				}(),
			},
			want: false,
		},
		{
			name: "different certfile",
			args: args{
				cfg1: cfg1,
				cfg2: func() tls.ClientConfig {
					c := cfg1
					c.CertFile = "file://path/to/othercert.pem"
					return c
				}(),
			},
			want: false,
		},
		{
			name: "different crl",
			args: args{
				cfg1: cfg1,
				cfg2: func() tls.ClientConfig {
					c := cfg1
					c.CRL = tls.CRLConfig{
						Enabled: true,
						HTTP:    &tls.HTTPConfig{},
					}
					return c
				}(),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.args.cfg1.Equals(tt.args.cfg2))
		})
	}
}
