package x509

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

// ReadFile reads certificates from file in PEM format
func ReadX509(path string) ([]*x509.Certificate, error) {
	certPEMBlock, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseX509(certPEMBlock)
}

// ParseX509 parses certificates from PEM format
func ParseX509(pemBlock []byte) ([]*x509.Certificate, error) {
	data := pemBlock
	var cas []*x509.Certificate
	for {
		certDERBlock, tmp := pem.Decode(data)
		if certDERBlock == nil {
			return nil, errors.New("cannot decode pem block")
		}
		certs, err := x509.ParseCertificates(certDERBlock.Bytes)
		if err != nil {
			return nil, err
		}
		cas = append(cas, certs...)
		if len(tmp) == 0 {
			break
		}
		data = tmp
	}
	return cas, nil
}

// ParsePrivateKey parses certificates from PEM format
func ParsePrivateKey(pemBlock []byte) (*ecdsa.PrivateKey, error) {
	certDERBlock, _ := pem.Decode(pemBlock)
	if certDERBlock == nil {
		return nil, errors.New("cannot decode pem block")
	}

	if key, err := x509.ParsePKCS8PrivateKey(certDERBlock.Bytes); err == nil {
		switch key := key.(type) {
		case *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("crypto/tls: found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(certDERBlock.Bytes); err == nil {
		return key, nil
	}

	return nil, errors.New("crypto/tls: failed to parse private key")
}

// ReadPrivateKey reads private key from file in PEM format
func ReadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	certPEMBlock, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParsePrivateKey(certPEMBlock)
}
