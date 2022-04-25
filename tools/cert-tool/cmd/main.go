package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/plgd-dev/kit/v2/security"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
)

type Options struct {
	Command struct {
		GenerateRootCA         bool   `long:"generateRootCA"`
		GenerateIntermediateCA bool   `long:"generateIntermediateCA"`
		GenerateCert           bool   `long:"generateCertificate"`
		GenerateIdentity       string `long:"generateIdentityCertificate" description:"deviceID"`
		GenerateIdentityCsr    string `long:"generateIdentityCsr" description:"deviceID"`
	} `group:"Command" namespace:"cmd"`
	Certificate generateCertificate.Configuration `group:"Certificate" namespace:"cert"`
	OutCert     string                            `long:"outCert" default:"cert.pem"`
	OutKey      string                            `long:"outKey" default:"cert.key"`
	OutCsr      string                            `long:"outCsr" default:"req.csr"`
	SignerCert  string                            `long:"signerCert"`
	SignerKey   string                            `long:"signerKey"`
}

func pemBlockForKey(k *ecdsa.PrivateKey) (*pem.Block, error) {
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return nil, err
	}
	return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
}

func cmdGenerateRootCA(opts Options, priv *ecdsa.PrivateKey) {
	cert, err := generateCertificate.GenerateRootCA(opts.Certificate, priv)
	if err != nil {
		log.Fatal(err)
	}
	writeCertOut(opts, cert)
	writePrivateKey(opts, priv)
}

func cmdGenerateIntermediateCA(opts Options, priv *ecdsa.PrivateKey) {
	signerCert, err := security.LoadX509(opts.SignerCert)
	if err != nil {
		log.Fatal(err)
	}
	signerKey, err := security.LoadX509PrivateKey(opts.SignerKey)
	if err != nil {
		log.Fatal(err)
	}
	cert, err := generateCertificate.GenerateIntermediateCA(opts.Certificate, priv, signerCert, signerKey)
	if err != nil {
		log.Fatal(err)
	}
	writeCertOut(opts, cert)
	writePrivateKey(opts, priv)
}

func cmdGenerateCert(opts Options, priv *ecdsa.PrivateKey) {
	signerCert, err := security.LoadX509(opts.SignerCert)
	if err != nil {
		log.Fatal(err)
	}
	signerKey, err := security.LoadX509PrivateKey(opts.SignerKey)
	if err != nil {
		log.Fatal(err)
	}
	cert, err := generateCertificate.GenerateCert(opts.Certificate, priv, signerCert, signerKey)
	if err != nil {
		log.Fatal(err)
	}
	writeCertOut(opts, cert)
	writePrivateKey(opts, priv)
}

func cmdGenerateIdentityCert(opts Options, priv *ecdsa.PrivateKey) {
	signerCert, err := security.LoadX509(opts.SignerCert)
	if err != nil {
		log.Fatal(err)
	}
	signerKey, err := security.LoadX509PrivateKey(opts.SignerKey)
	if err != nil {
		log.Fatal(err)
	}
	cert, err := generateCertificate.GenerateIdentityCert(opts.Certificate, opts.Command.GenerateIdentity, priv, signerCert, signerKey)
	if err != nil {
		log.Fatal(err)
	}
	writeCertOut(opts, cert)
	writePrivateKey(opts, priv)
}

func cmdGenerateIdentityCSR(opts Options, priv *ecdsa.PrivateKey) {
	csr, err := generateCertificate.GenerateIdentityCSR(opts.Certificate, opts.Command.GenerateIdentityCsr, priv)
	if err != nil {
		log.Fatal(err)
	}
	writePrivateKey(opts, priv)
	writeCsrOut(opts, csr)
}

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()

	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	switch {
	case opts.Command.GenerateRootCA:
		cmdGenerateRootCA(opts, priv)
	case opts.Command.GenerateIntermediateCA:
		cmdGenerateIntermediateCA(opts, priv)
	case opts.Command.GenerateCert:
		cmdGenerateCert(opts, priv)
	case opts.Command.GenerateIdentity != "":
		cmdGenerateIdentityCert(opts, priv)
	case opts.Command.GenerateIdentityCsr != "":
		cmdGenerateIdentityCSR(opts, priv)
	default:
		fmt.Println("invalid command")
		parser.WriteHelp(os.Stdout)
		os.Exit(2)
	}
}

func writeToFile(file string, data []byte) {
	certOut, err := os.Create(file)
	if err != nil {
		log.Fatalf("failed to open %v: %s", file, err)
	}
	_, err = certOut.Write(data)
	if err != nil {
		log.Fatalf("failed to write %v: %s", file, err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("error closing %v: %s", file, err)
	}
}

func writeCertOut(opts Options, cert []byte) {
	writeToFile(opts.OutCert, cert)
}

func writeCsrOut(opts Options, csr []byte) {
	writeToFile(opts.OutCsr, csr)
}

func writePrivateKey(opts Options, priv *ecdsa.PrivateKey) {
	privBlock, err := pemBlockForKey(priv)
	if err != nil {
		log.Fatalf("failed to encode priv key %v: %v", opts.OutKey, err)
	}
	writeToFile(opts.OutKey, pem.EncodeToMemory(privBlock))
}
