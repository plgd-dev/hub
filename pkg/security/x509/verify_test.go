package x509_test

import (
	"crypto/x509"
	"testing"

	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	testX509 "github.com/plgd-dev/hub/v2/test/security/x509"
	"github.com/stretchr/testify/require"
)

func TestVerify(t *testing.T) {
	rootCert, rootPrivKey := testX509.CreateCACertificate(t)
	intermediateCert1, intermediatePrivKey1 := testX509.CreateIntermediateCACertificate(t, rootCert, rootPrivKey)
	intermediateCert2, _ := testX509.CreateIntermediateCACertificate(t, intermediateCert1, intermediatePrivKey1)

	rootCert1, _ := testX509.CreateCACertificate(t)
	rootCert1x509, err := pkgX509.ParseX509(rootCert1)
	require.NoError(t, err)

	intermediateCert2x509, err := pkgX509.ParseX509(intermediateCert2)
	require.NoError(t, err)

	type args struct {
		certificates           []*x509.Certificate
		certificateAuthorities []*x509.Certificate
		useSystemRoots         bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "empty-fail",
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				certificates:           intermediateCert2x509,
				certificateAuthorities: intermediateCert2x509,
			},
		},
		{
			name: "valid-with-system-roots",
			args: args{
				certificates:           intermediateCert2x509,
				certificateAuthorities: intermediateCert2x509,
				useSystemRoots:         true,
			},
		},
		{
			name: "empty-root-ca-fail",
			args: args{
				certificates: intermediateCert2x509,
			},
			wantErr: true,
		},
		{
			name: "empty-certs-fail",
			args: args{
				certificateAuthorities: intermediateCert2x509,
			},
			wantErr: true,
		},
		{
			name: "different-root-ca-fail",
			args: args{
				certificateAuthorities: intermediateCert2x509,
				certificates:           rootCert1x509,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err = pkgX509.Verify(tt.args.certificates, tt.args.certificateAuthorities, tt.args.useSystemRoots, x509.VerifyOptions{})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
