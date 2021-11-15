package service

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"
)

func parseUUID(commonName string) string {
	m := regexp.MustCompile("[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")
	return m.FindString(commonName)
}

type infoData struct {
	CertificateCommonName   string
	CertificateCommonNameID string
}

func (d infoData) String() string {
	return fmt.Sprintf("certificateCommonName=%v, certificateCommonNameID=%v", d.CertificateCommonName, d.CertificateCommonNameID)
}

// PrepareSignRequest can create a CertificateRequest struct with all the required data filled
func getInfoData(ctx context.Context, csr []byte) (infoData, error) {
	csrBlock, _ := pem.Decode(csr)
	if csrBlock == nil {
		return infoData{}, fmt.Errorf("pem not found")
	}

	certificateRequest, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		return infoData{}, err
	}

	err = certificateRequest.CheckSignature()
	if err != nil {
		return infoData{}, err
	}

	return infoData{
		CertificateCommonName:   certificateRequest.Subject.CommonName,
		CertificateCommonNameID: parseUUID(certificateRequest.Subject.CommonName),
	}, nil
}
