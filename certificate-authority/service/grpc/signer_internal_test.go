package grpc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/kit/v2/security"
	"github.com/stretchr/testify/require"
)

func privateKeyToPem(t *testing.T, priv *ecdsa.PrivateKey) []byte {
	privKey, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privKey})
}

func createCACertificate(t *testing.T) ([]byte, *ecdsa.PrivateKey) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = "rootCA"
	cfg.ValidFor = time.Hour * 24
	cfg.BasicConstraints.MaxPathLen = 1000
	rootCA, err := generateCertificate.GenerateRootCA(cfg, priv)
	require.NoError(t, err)
	return rootCA, priv
}

func createIntermediateCACertificate(t *testing.T, signerCerts []byte, signerPriv *ecdsa.PrivateKey) ([]byte, *ecdsa.PrivateKey) {
	certs, err := security.ParseX509FromPEM(signerCerts)
	require.NoError(t, err)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = "intermediateCA"
	cfg.ValidFor = time.Hour * 24
	cfg.BasicConstraints.MaxPathLen = 100
	rootCA, err := generateCertificate.GenerateIntermediateCA(cfg, priv, certs, signerPriv)
	require.NoError(t, err)
	return rootCA, priv
}

func joinPems(pems ...[]byte) []byte {
	ret := make([]byte, 0, 4096)
	for i := range pems {
		ret = append(ret, pems[i]...)
	}
	return ret
}

func certificatesToPems(certs []*x509.Certificate) []byte {
	ret := make([]byte, 0, 4096)
	for i := range certs {
		ret = append(ret, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certs[i].Raw})...)
	}
	return ret
}

func getLeafCertificate(pem []byte) []byte {
	certs, err := security.ParseX509FromPEM(pem)
	if err != nil {
		panic(err)
	}
	return certificatesToPems(certs[:1])
}

func TestNewSigner(t *testing.T) {
	tmp, err := os.MkdirTemp("", "testNewSigner*****")
	require.NoError(t, err)
	defer func() {
		err = os.RemoveAll(tmp)
		require.NoError(t, err)
	}()
	rootCert, rootPrivKey := createCACertificate(t)
	intermediateCert1, intermediatePrivKey1 := createIntermediateCACertificate(t, rootCert, rootPrivKey)
	intermediateCert2, intermediatePrivKey2 := createIntermediateCACertificate(t, intermediateCert1, intermediatePrivKey1)

	intermediateCert1 = getLeafCertificate(intermediateCert1)
	intermediateCert2 = getLeafCertificate(intermediateCert2)

	fullChainCrt := path.Join(tmp, "fullChain.crt")
	rootIntermediate1Crt := path.Join(tmp, "rootIntermediate1.crt")
	intermediate2Crt := path.Join(tmp, "intermediate2.crt")
	intermediate2intermediate1 := path.Join(tmp, "intermediate2intermediate1.crt")
	intermediate2Key := path.Join(tmp, "intermediate2.key")
	rootCrt := path.Join(tmp, "root.crt")
	rootKey := path.Join(tmp, "root.key")

	err = os.WriteFile(fullChainCrt, joinPems(intermediateCert2, intermediateCert1, rootCert), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(rootIntermediate1Crt, joinPems(intermediateCert1, rootCert), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(intermediate2Crt, intermediateCert2, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(intermediate2intermediate1, joinPems(intermediateCert2, intermediateCert1), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(intermediate2Key, privateKeyToPem(t, intermediatePrivKey2), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(rootCrt, rootCert, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(rootKey, privateKeyToPem(t, rootPrivKey), 0o600)
	require.NoError(t, err)
	type args struct {
		signerConfig SignerConfig
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
					CertFile: rootCrt,
					KeyFile:  rootKey,
				},
			},
			wantCertificates: joinPems(rootCert),
		},
		{
			name: "fullChain",
			args: args{
				signerConfig: SignerConfig{
					CertFile: fullChainCrt,
					KeyFile:  intermediate2Key,
				},
			},
			wantCertificates: joinPems(intermediateCert2, intermediateCert1, rootCert),
		},
		{
			name: "intermediate2Crt",
			args: args{
				signerConfig: SignerConfig{
					caPoolArray: []string{rootIntermediate1Crt},
					CertFile:    intermediate2Crt,
					KeyFile:     intermediate2Key,
				},
			},
			wantCertificates: joinPems(intermediateCert2, intermediateCert1, rootCert),
		},
		{
			name: "intermediate2intermediate1",
			args: args{
				signerConfig: SignerConfig{
					caPoolArray: []string{rootCrt},
					CertFile:    intermediate2intermediate1,
					KeyFile:     intermediate2Key,
				},
			},
			wantCertificates: joinPems(intermediateCert2, intermediateCert1, rootCert),
		},
		{
			name: "intermediate2Crt - fail",
			args: args{
				signerConfig: SignerConfig{
					caPoolArray: []string{rootCrt},
					CertFile:    intermediate2Crt,
					KeyFile:     intermediate2Key,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSigner("tt.args.ownerClaim", "tt.args.hubID", tt.args.signerConfig)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
			require.Equal(t, string(tt.wantCertificates), string(certificatesToPems(got.certificate)))
		})
	}
}
