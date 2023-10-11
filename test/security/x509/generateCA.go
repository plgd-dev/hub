package x509

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	"github.com/stretchr/testify/require"
)

func PrivateKeyToPem(t *testing.T, priv *ecdsa.PrivateKey) []byte {
	privKey, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privKey})
}

func CreateCACertificate(t *testing.T) ([]byte, *ecdsa.PrivateKey) {
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

func CreateIntermediateCACertificate(t *testing.T, signerCerts []byte, signerPriv *ecdsa.PrivateKey) ([]byte, *ecdsa.PrivateKey) {
	certs, err := pkgX509.ParseX509(signerCerts)
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

func JoinPems(pems ...[]byte) []byte {
	ret := make([]byte, 0, 4096)
	for i := range pems {
		ret = append(ret, pems[i]...)
	}
	return ret
}

func CertificatesToPems(certs []*x509.Certificate) []byte {
	ret := make([]byte, 0, 4096)
	for i := range certs {
		ret = append(ret, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certs[i].Raw})...)
	}
	return ret
}

func GetLeafCertificate(pem []byte) []byte {
	certs, err := pkgX509.ParseX509(pem)
	if err != nil {
		panic(err)
	}
	return CertificatesToPems(certs[:1])
}
