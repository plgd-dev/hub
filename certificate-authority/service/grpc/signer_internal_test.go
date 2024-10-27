package grpc

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/test/security/x509"
	"github.com/stretchr/testify/require"
)

func TestNewSigner(t *testing.T) {
	tmp, err := os.MkdirTemp("", "testNewSigner*****")
	require.NoError(t, err)
	defer func() {
		err = os.RemoveAll(tmp)
		require.NoError(t, err)
	}()
	rootCert, rootPrivKey := x509.CreateCACertificate(t)
	intermediateCert1, intermediatePrivKey1 := x509.CreateIntermediateCACertificate(t, rootCert, rootPrivKey)
	intermediateCert2, intermediatePrivKey2 := x509.CreateIntermediateCACertificate(t, intermediateCert1, intermediatePrivKey1)

	intermediateCert1 = x509.GetLeafCertificate(intermediateCert1)
	intermediateCert2 = x509.GetLeafCertificate(intermediateCert2)

	fullChainCrt := path.Join(tmp, "fullChain.crt")
	rootIntermediate1Crt := path.Join(tmp, "rootIntermediate1.crt")
	intermediate2Crt := path.Join(tmp, "intermediate2.crt")
	intermediate2intermediate1 := path.Join(tmp, "intermediate2intermediate1.crt")
	intermediate2Key := path.Join(tmp, "intermediate2.key")
	rootCrt := path.Join(tmp, "root.crt")
	rootKey := path.Join(tmp, "root.key")

	err = os.WriteFile(fullChainCrt, x509.JoinPems(intermediateCert2, intermediateCert1, rootCert), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(rootIntermediate1Crt, x509.JoinPems(intermediateCert1, rootCert), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(intermediate2Crt, intermediateCert2, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(intermediate2intermediate1, x509.JoinPems(intermediateCert2, intermediateCert1), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(intermediate2Key, x509.PrivateKeyToPem(t, intermediatePrivKey2), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(rootCrt, rootCert, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(rootKey, x509.PrivateKeyToPem(t, rootPrivKey), 0o600)
	require.NoError(t, err)
	type args struct {
		signerConfig SignerConfig
		crlServer    string
	}
	tests := []struct {
		name             string
		args             args
		wantErr          bool
		wantCertificates []byte
	}{
		{
			name:    "empty",
			wantErr: true,
		},
		{
			name: "root",
			args: args{
				signerConfig: SignerConfig{
					CertFile: urischeme.URIScheme(rootCrt),
					KeyFile:  urischeme.URIScheme(rootKey),
				},
			},
			wantCertificates: x509.JoinPems(rootCert),
		},
		{
			name: "fullChain",
			args: args{
				signerConfig: SignerConfig{
					CertFile: urischeme.URIScheme(fullChainCrt),
					KeyFile:  urischeme.URIScheme(intermediate2Key),
				},
			},
			wantCertificates: x509.JoinPems(intermediateCert2, intermediateCert1, rootCert),
		},
		{
			name: "intermediate2Crt",
			args: args{
				signerConfig: SignerConfig{
					caPoolArray: []urischeme.URIScheme{urischeme.URIScheme(rootIntermediate1Crt)},
					CertFile:    urischeme.URIScheme(intermediate2Crt),
					KeyFile:     urischeme.URIScheme(intermediate2Key),
				},
			},
			wantCertificates: x509.JoinPems(intermediateCert2, intermediateCert1, rootCert),
		},
		{
			name: "intermediate2intermediate1",
			args: args{
				signerConfig: SignerConfig{
					caPoolArray: []urischeme.URIScheme{urischeme.URIScheme(rootCrt)},
					CertFile:    urischeme.URIScheme(intermediate2intermediate1),
					KeyFile:     urischeme.URIScheme(intermediate2Key),
				},
			},
			wantCertificates: x509.JoinPems(intermediateCert2, intermediateCert1, rootCert),
		},
		{
			name: "intermediate2Crt - fail",
			args: args{
				signerConfig: SignerConfig{
					caPoolArray: []urischeme.URIScheme{urischeme.URIScheme(rootCrt)},
					CertFile:    urischeme.URIScheme(intermediate2Crt),
					KeyFile:     urischeme.URIScheme(intermediate2Key),
				},
			},
			wantErr: true,
		},
		{
			name: "with crl server",
			args: args{
				signerConfig: SignerConfig{
					CertFile:  urischeme.URIScheme(fullChainCrt),
					KeyFile:   urischeme.URIScheme(intermediate2Key),
					ValidFrom: "2001-01-01T00:00:00Z",
					ExpiresIn: time.Hour,
					CRL: CRLConfig{
						Enabled:   true,
						ExpiresIn: time.Hour,
					},
				},
				crlServer: "https://crl.example.com",
			},
			wantCertificates: x509.JoinPems(intermediateCert2, intermediateCert1, rootCert),
		},
		{
			name: "with crl server - fail",
			args: args{
				signerConfig: SignerConfig{
					CertFile:  urischeme.URIScheme(fullChainCrt),
					KeyFile:   urischeme.URIScheme(intermediate2Key),
					ValidFrom: "2001-01-01T00:00:00Z",
					ExpiresIn: time.Hour,
					CRL: CRLConfig{
						Enabled:   true,
						ExpiresIn: time.Hour,
					},
				},
				crlServer: "not-an-URL",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSigner("ownerClaim", "hubID", tt.args.crlServer, tt.args.signerConfig)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
			require.Equal(t, string(tt.wantCertificates), string(x509.CertificatesToPems(got.certificate)))
		})
	}
}
