package service

import (
	"context"
	"crypto/x509"
	"errors"

	"github.com/smallstep/certificates/authority/provisioner"
)

// ACME is the acme provisioner type, an entity that can authorize the ACME
// provisioning flow.
type ACME struct {
	ProvisionerName ProvisionerName
}

// ProvisionerName provides name of provision
type ProvisionerName string

// GetID returns the provisioner unique identifier.
func (p ACME) GetID() string {
	return "acme/" + string(p.ProvisionerName)
}

// GetTokenID returns the identifier of the token.
func (p ACME) GetTokenID(ott string) (string, error) {
	return "", errors.New("acme provisioner does not implement GetTokenID")
}

// GetName returns the name of the provisioner.
func (p ACME) GetName() string {
	return string(p.ProvisionerName)
}

// GetType returns the type of provisioner.
func (p ACME) GetType() provisioner.Type {
	return provisioner.TypeACME
}

// GetEncryptedKey returns the base provisioner encrypted key if it's defined.
func (p ACME) GetEncryptedKey() (string, string, bool) {
	return "", "", false
}

// Init initializes and validates the fields of a JWK type.
func (ACME) Init(config provisioner.Config) error {
	return nil
}

// AuthorizeSign validates the given token.
func (a ACME) AuthorizeSign(ctx context.Context, token string) ([]provisioner.SignOption, error) {
	return []provisioner.SignOption{
		a.ProvisionerName,
		ctx,
	}, nil
}

// AuthorizeRenewal is not implemented for the ACME provisioner.
func (ACME) AuthorizeRenewal(cert *x509.Certificate) error {
	return nil
}

// AuthorizeRevoke is not implemented yet for the ACME provisioner.
func (ACME) AuthorizeRevoke(token string) error {
	return nil
}
