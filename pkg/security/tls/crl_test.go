package tls_test

import (
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/security/tls"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestHTTPConfigValidate(t *testing.T) {
	validHTTPConfig := func() *tls.HTTPConfig {
		return &tls.HTTPConfig{
			MaxIdleConns:        10,
			MaxConnsPerHost:     5,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     10 * time.Second,
			Timeout:             5 * time.Second,
			TLS: tls.ClientConfig{
				CAPool:   []string{"file://path/to/ca1.pem", "file://path/to/ca2.pem"},
				KeyFile:  "file://key.pem",
				CertFile: "file://cert.pem",
			},
		}
	}

	type args struct {
		cfg *tls.HTTPConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid configuration",
			args: args{
				cfg: validHTTPConfig(),
			},
		},
		{
			name: "negative MaxIdleConns",
			args: args{
				cfg: func() *tls.HTTPConfig {
					cfg := validHTTPConfig()
					cfg.MaxIdleConns = -1
					return cfg
				}(),
			},
			wantErr: true,
		},
		{
			name: "negative MaxConnsPerHost",
			args: args{
				cfg: func() *tls.HTTPConfig {
					cfg := validHTTPConfig()
					cfg.MaxConnsPerHost = -1
					return cfg
				}(),
			},
			wantErr: true,
		},
		{
			name: "negative MaxIdleConnsPerHost",
			args: args{
				cfg: func() *tls.HTTPConfig {
					cfg := validHTTPConfig()
					cfg.MaxIdleConnsPerHost = -1
					return cfg
				}(),
			},
			wantErr: true,
		},
		{
			name: "negative IdleConnTimeout",
			args: args{
				cfg: func() *tls.HTTPConfig {
					cfg := validHTTPConfig()
					cfg.IdleConnTimeout = -1
					return cfg
				}(),
			},
			wantErr: true,
		},
		{
			name: "negative Timeout",
			args: args{
				cfg: func() *tls.HTTPConfig {
					cfg := validHTTPConfig()
					cfg.Timeout = -1
					return cfg
				}(),
			},
			wantErr: true,
		},
		{
			name: "crl invalid - missing http",
			args: args{
				cfg: func() *tls.HTTPConfig {
					cfg := validHTTPConfig()
					cfg.TLS.CRL.Enabled = true
					return cfg
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

func TestCRLConfigValidate(t *testing.T) {
	validCRLConfig := func() *tls.CRLConfig {
		return &tls.CRLConfig{
			Enabled: true,
			HTTP: &tls.HTTPConfig{
				MaxIdleConns:    10,
				MaxConnsPerHost: 5,
				TLS: tls.ClientConfig{
					CAPool:   []string{"file://path/to/ca1.pem", "file://path/to/ca2.pem"},
					KeyFile:  "file://key.pem",
					CertFile: "file://cert.pem",
				},
			},
		}
	}

	type args struct {
		cfg *tls.CRLConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid configuration",
			args: args{
				cfg: validCRLConfig(),
			},
		},
		{
			name: "enabled but missing HTTP",
			args: args{
				cfg: &tls.CRLConfig{Enabled: true},
			},
			wantErr: true,
		},
		{
			name: "disabled, missing HTTP",
			args: args{
				cfg: &tls.CRLConfig{Enabled: false},
			},
			wantErr: false,
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

func TestCRLConfigEquals(t *testing.T) {
	cfg1 := func() tls.CRLConfig {
		return tls.CRLConfig{
			Enabled: true,
			HTTP: &tls.HTTPConfig{
				MaxIdleConns:    10,
				MaxConnsPerHost: 5,
				TLS: tls.ClientConfig{
					KeyFile:  "file://key.pem",
					CertFile: "file://cert.pem",
				},
			},
		}
	}

	type args struct {
		c1 tls.CRLConfig
		c2 tls.CRLConfig
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal configurations",
			args: args{
				c1: cfg1(),
				c2: cfg1(),
			},
			want: true,
		},
		{
			name: "different Enabled field",
			args: args{
				c1: cfg1(),
				c2: func() tls.CRLConfig {
					c := cfg1()
					c.Enabled = false
					return c
				}(),
			},
			want: false,
		},
		{
			name: "HTTP config is nil in one",
			args: args{
				c1: cfg1(),
				c2: func() tls.CRLConfig {
					c := cfg1()
					c.HTTP = nil
					return c
				}(),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.c1.Equals(tt.args.c2)
			require.Equal(t, tt.want, got)
		})
	}
}

func marshal(t *testing.T, in interface{}) string {
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	return string(out)
}

func TestCRLConfigUnmarshalYAML(t *testing.T) {
	type args struct {
		yaml string
	}
	tests := []struct {
		name    string
		args    args
		want    tls.CRLConfig
		wantErr bool
	}{
		{
			name: "valid - CRL disabled",
			args: args{
				yaml: `enabled: false`,
			},
			want: tls.CRLConfig{
				Enabled: false,
				HTTP:    nil,
			},
		},
		{
			name: "valid - CRL enabled",
			args: args{
				yaml: `enabled: true
http:
  maxIdleConns: 10
  maxConnsPerHost: 20
  maxIdleConnsPerHost: 5
  idleConnTimeout: 30s
  timeout: 60s
`,
			},
			want: tls.CRLConfig{
				Enabled: true,
				HTTP: &tls.HTTPConfig{
					MaxIdleConns:        10,
					MaxConnsPerHost:     20,
					MaxIdleConnsPerHost: 5,
					IdleConnTimeout:     30 * time.Second,
					Timeout:             60 * time.Second,
				},
			},
		},
		{
			name: "valid - CRL enabled, HTTP with TLS",
			args: args{
				yaml: `enabled: true
http:
  maxIdleConns: 10
  maxConnsPerHost: 20
  maxIdleConnsPerHost: 5
  idleConnTimeout: 30s
  timeout: 60s
  tls:
    caPool: /capool
    keyFile: /keyfile
    certFile: /certfile
    useSystemCAPool: true
`,
			},
			want: tls.CRLConfig{
				Enabled: true,
				HTTP: &tls.HTTPConfig{
					MaxIdleConns:        10,
					MaxConnsPerHost:     20,
					MaxIdleConnsPerHost: 5,
					IdleConnTimeout:     30 * time.Second,
					Timeout:             60 * time.Second,
					TLS: tls.ClientConfig{
						CAPool:          "/capool",
						KeyFile:         "/keyfile",
						CertFile:        "/certfile",
						UseSystemCAPool: true,
					},
				},
			},
		},
		{
			name: "valid - CRL enabled, HTTP with TLS, recursive",
			args: args{
				yaml: `enabled: true
http:
  maxIdleConns: 10
  maxConnsPerHost: 20
  maxIdleConnsPerHost: 5
  idleConnTimeout: 30s
  timeout: 60s
  tls:
    caPool: /capool
    keyFile: /keyfile
    certFile: /certfile
    useSystemCAPool: true
    crl:
      enabled: true
      http:
        maxIdleConns: 20
        maxConnsPerHost: 40
        maxIdleConnsPerHost: 10
        idleConnTimeout: 60s
        timeout: 120s
`,
			},
			want: tls.CRLConfig{
				Enabled: true,
				HTTP: &tls.HTTPConfig{
					MaxIdleConns:        10,
					MaxConnsPerHost:     20,
					MaxIdleConnsPerHost: 5,
					IdleConnTimeout:     30 * time.Second,
					Timeout:             60 * time.Second,
					TLS: tls.ClientConfig{
						CAPool:          "/capool",
						KeyFile:         "/keyfile",
						CertFile:        "/certfile",
						UseSystemCAPool: true,
						CRL: tls.CRLConfig{
							Enabled: true,
							HTTP: &tls.HTTPConfig{
								MaxIdleConns:        20,
								MaxConnsPerHost:     40,
								MaxIdleConnsPerHost: 10,
								IdleConnTimeout:     60 * time.Second,
								Timeout:             120 * time.Second,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var crlConfig tls.CRLConfig
			err := yaml.Unmarshal([]byte(tt.args.yaml), &crlConfig)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.True(t, tt.want.Equals(crlConfig), "want:\n\n%v\nactual:\n\n%v\n", marshal(t, tt.want), marshal(t, crlConfig))
		})
	}
}
