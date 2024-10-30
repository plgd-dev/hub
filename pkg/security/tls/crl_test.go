package tls_test

import (
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/security/tls"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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
