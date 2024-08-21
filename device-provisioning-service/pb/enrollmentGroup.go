package pb

import (
	"crypto/x509"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	"github.com/plgd-dev/kit/v2/security"
)

type EnrollmentGroups []*EnrollmentGroup

func (p EnrollmentGroups) Sort() {
	sort.Slice(p, func(i, j int) bool {
		return p[i].GetId() < p[j].GetId()
	})
}

func checkLeadCertificateNameInChains(leadCertificateName string, chains [][]*x509.Certificate) bool {
	for _, chain := range chains {
		for _, cert := range chain {
			if cert.Subject.CommonName == leadCertificateName {
				return true
			}
		}
	}
	return false
}

func (c *EnrollmentGroup) ResolvePreSharedKey() (string, bool, error) {
	if c.GetPreSharedKey() == "" {
		return "", false, nil
	}
	data, err := urischeme.URIScheme(c.GetPreSharedKey()).Read()
	if err != nil {
		return "", false, err
	}
	if len(data) < 16 {
		return "", false, fmt.Errorf("at least 16 bytes are required, but %v bytes are provided", len(data))
	}
	return string(data), true, nil
}

func (c *X509Configuration) ResolveCertificateChain() ([]*x509.Certificate, error) {
	data, err := urischeme.URIScheme(c.GetCertificateChain()).Read()
	if err != nil {
		return nil, fmt.Errorf("cannot read certificateChain('%v') - %w", c.GetCertificateChain(), err)
	}
	return security.ParseX509FromPEM(data)
}

func (c *X509Configuration) Validate() error {
	if c.GetCertificateChain() == "" {
		return fmt.Errorf("certificateChain('%v')", c.GetCertificateChain())
	}
	chain, err := c.ResolveCertificateChain()
	if err != nil {
		return fmt.Errorf("certificateChain('%v') - %w", c.GetCertificateChain(), err)
	}
	verifiedChains, err := pkgX509.Verify(chain, chain, false, x509.VerifyOptions{})
	if err != nil {
		return fmt.Errorf("certificateChain('%v') - %w", c.GetCertificateChain(), err)
	}
	if c.GetLeadCertificateName() == "" {
		c.LeadCertificateName = chain[0].Subject.CommonName
	} else if !checkLeadCertificateNameInChains(c.GetLeadCertificateName(), verifiedChains) {
		return fmt.Errorf("leadCertificateName('%v') - not found in certificateChain", c.GetLeadCertificateName())
	}

	return nil
}

func (c *AttestationMechanism) Validate() error {
	if c.GetX509() == nil {
		return errors.New("x509 - is empty")
	}
	if err := c.GetX509().Validate(); err != nil {
		return fmt.Errorf("x509.%w", err)
	}
	return nil
}

func (c *EnrollmentGroup) Validate(owner string) error {
	if _, err := uuid.Parse(c.GetId()); err != nil {
		return fmt.Errorf("id('%v') - %w", c.GetId(), err)
	}
	if c.GetOwner() == "" {
		return fmt.Errorf("owner('%v') - is empty", c.GetOwner())
	}
	if owner != "" && owner != c.GetOwner() {
		return fmt.Errorf("owner('%v') - expects %v", c.GetOwner(), owner)
	}
	_, _, err := c.ResolvePreSharedKey()
	if err != nil {
		return fmt.Errorf("preSharedKey('%v') - %w", c.GetPreSharedKey(), err)
	}
	if c.GetAttestationMechanism() == nil {
		return errors.New("attestationMechanism - is empty")
	}
	if err := c.GetAttestationMechanism().Validate(); err != nil {
		return fmt.Errorf("attestationMechanism.%w", err)
	}
	if c.GetName() == "" {
		// set default name as id for backward compatibility
		c.Name = c.GetId()
	}
	if len(c.GetHubIds()) == 0 {
		return errors.New("hubIds - is empty")
	}
	for idx, hubID := range c.GetHubIds() {
		if _, err := uuid.Parse(hubID); err != nil {
			return fmt.Errorf("hubIds[%v]('%v') - %w", idx, hubID, err)
		}
	}
	return nil
}
