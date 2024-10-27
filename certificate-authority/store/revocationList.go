package store

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

const (
	CertificatesKey = "certificates" // must match with RevocationList.Certificates bson tag
	IssuedAtKey     = "issuedAt"     // must match with RevocationListCertificate.IssuedAt bson tag
	NumberKey       = "number"       // must match with RevocationListCertificate.NumberKey bson tag
	SerialKey       = "serial"       // must match with RevocationListCertificate.Serial bson tag
	ValidUntilKey   = "validUntil"   // must match with RevocationListCertificate.ValidUntil bson tag
	RevocationKey   = "revocation"   // must match with RevocationListCertificate.Revocation bson tag
)

type RevocationListCertificate struct {
	// Serial number
	Serial string `bson:"serial"`
	// Time until the record is valid in Unix nanoseconds timestamp format
	ValidUntil int64 `bson:"validUntil,omitempty"`
	// Revocation time of the certificate in Unix nanoseconds timestamp format.
	Revocation int64 `bson:"revocation"`
}

func (rlc *RevocationListCertificate) Validate() error {
	if rlc.Serial == "" {
		return errors.New("serial number not set")
	}
	if rlc.Revocation == 0 {
		return errors.New("revocation time not set")
	}
	return nil
}

type RevocationList struct {
	// The record ID is determined by applying a formula that utilizes the public key of the issuer, and it is computed as uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw).
	Id string `bson:"_id"`
	// Number is used to populate the X.509 v2 cRLNumber extension in the CRL, which should be a monotonically increasing sequence number for a given
	// CRL scope and CRL issuer.
	Number string `bson:"number"`
	// Time when the CRL was issued in Unix nanoseconds timestamp format
	IssuedAt int64 `bson:"issuedAt"`
	// Time until the issued CRL is valid in Unix nanoseconds timestamp format
	ValidUntil int64 `bson:"validUntil"`
	// List of revoked certificates issued by the issuer
	Certificates []*RevocationListCertificate `bson:"certificates,omitempty"`
}

func ParseBigInt(s string) (*big.Int, error) {
	var number big.Int
	if _, ok := number.SetString(s, 10); !ok {
		return nil, fmt.Errorf("invalid numeric string(%v)", s)
	}
	return &number, nil
}

// TODO: use some delta to check expiration
func (rl *RevocationList) IsExpired() bool {
	return rl.ValidUntil <= time.Now().UnixNano()
}

func (rl *RevocationList) Validate() error {
	if _, err := uuid.Parse(rl.Id); err != nil {
		return fmt.Errorf("invalid ID(%v): %w", rl.Id, err)
	}
	if (rl.IssuedAt == 0 && rl.ValidUntil != 0) || (rl.ValidUntil < rl.IssuedAt) {
		return fmt.Errorf("invalid validity period timestamps(from %v to %v)", rl.IssuedAt, rl.ValidUntil)
	}
	if _, err := ParseBigInt(rl.Number); err != nil {
		return err
	}
	for _, c := range rl.Certificates {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	return nil
}
