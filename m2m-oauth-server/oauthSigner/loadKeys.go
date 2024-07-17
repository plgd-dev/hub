package oauthsigner

import (
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
)

func LoadPrivateKey(path urischeme.URIScheme) (interface{}, error) {
	certPEMBlock, err := path.Read()
	if err != nil {
		return nil, err
	}
	certDERBlock, _ := pem.Decode(certPEMBlock)
	if certDERBlock == nil {
		return nil, errors.New("cannot decode pem block")
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
	return nil, errors.New("unknown type")
}
