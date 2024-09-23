package grpc_test

import (
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/stretchr/testify/require"
)

func TestCRLConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   grpc.CRLConfig
		wantErr bool
	}{
		{
			name: "Disabled CRLConfig",
			input: grpc.CRLConfig{
				Enabled: false,
			},
		},
		{
			name: "Enabled CRLConfig with valid ExternalAddress and ExpiresIn",
			input: grpc.CRLConfig{
				Enabled:   true,
				ExpiresIn: time.Hour,
			},
		},
		{
			name: "Enabled CRLConfig with ExpiresIn less than 1 minute",
			input: grpc.CRLConfig{
				Enabled:   true,
				ExpiresIn: 30 * time.Second,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSignerConfigValidate(t *testing.T) {
	crl := grpc.CRLConfig{
		Enabled:   true,
		ExpiresIn: time.Hour,
	}
	tests := []struct {
		name    string
		input   grpc.SignerConfig
		wantErr bool
	}{
		{
			name: "Valid SignerConfig",
			input: grpc.SignerConfig{
				CAPool:    []string{"ca1.pem", "ca2.pem"},
				KeyFile:   urischeme.URIScheme("key.pem"),
				CertFile:  urischeme.URIScheme("cert.pem"),
				ValidFrom: time.Now().Format(time.RFC3339),
				ExpiresIn: time.Hour * 24,
				CRL:       crl,
			},
		},
		{
			name: "Invalid CA Pool",
			input: grpc.SignerConfig{
				CAPool:    42,
				KeyFile:   urischeme.URIScheme("key.pem"),
				CertFile:  urischeme.URIScheme("cert.pem"),
				ValidFrom: time.Now().Format(time.RFC3339),
				ExpiresIn: time.Hour * 24,
				CRL:       crl,
			},
			wantErr: true,
		},
		{
			name: "Empty CertFile",
			input: grpc.SignerConfig{
				CAPool:    []string{"ca1.pem"},
				KeyFile:   urischeme.URIScheme("key.pem"),
				CertFile:  "",
				ValidFrom: time.Now().Format(time.RFC3339),
				ExpiresIn: time.Hour * 24,
				CRL:       crl,
			},
			wantErr: true,
		},
		{
			name: "Empty KeyFile",
			input: grpc.SignerConfig{
				CAPool:    []string{"ca1.pem"},
				KeyFile:   "",
				CertFile:  urischeme.URIScheme("cert.pem"),
				ValidFrom: time.Now().Format(time.RFC3339),
				ExpiresIn: time.Hour * 24,
				CRL:       crl,
			},
			wantErr: true,
		},
		{
			name: "Invalid ExpiresIn",
			input: grpc.SignerConfig{
				CAPool:    []string{"ca1.pem", "ca2.pem"},
				KeyFile:   urischeme.URIScheme("key.pem"),
				CertFile:  urischeme.URIScheme("cert.pem"),
				ValidFrom: time.Now().Format(time.RFC3339),
				ExpiresIn: -1,
				CRL:       crl,
			},
			wantErr: true,
		},
		{
			name: "Invalid ValidFrom format",
			input: grpc.SignerConfig{
				CAPool:    []string{"ca1.pem"},
				KeyFile:   urischeme.URIScheme("key.pem"),
				CertFile:  urischeme.URIScheme("cert.pem"),
				ValidFrom: "invalid-date",
				ExpiresIn: time.Hour * 24,
				CRL:       crl,
			},
			wantErr: true,
		},
		{
			name: "Invalid CRL",
			input: grpc.SignerConfig{
				CAPool:    []string{"ca1.pem", "ca2.pem"},
				KeyFile:   urischeme.URIScheme("key.pem"),
				CertFile:  urischeme.URIScheme("cert.pem"),
				ValidFrom: time.Now().Format(time.RFC3339),
				ExpiresIn: time.Hour * 24,
				CRL: grpc.CRLConfig{
					Enabled: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
