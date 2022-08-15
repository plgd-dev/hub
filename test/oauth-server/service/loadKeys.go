package service

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func LoadPrivateKey(path string) (interface{}, error) {
	certPEMBlock, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	certDERBlock, _ := pem.Decode(certPEMBlock)
	if certDERBlock == nil {
		return nil, fmt.Errorf("cannot decode pem block")
	}

	if key, err := x509.ParsePKCS8PrivateKey(certDERBlock.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(certDERBlock.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(certDERBlock.Bytes); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("unknown type")
}
