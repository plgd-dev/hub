package pb

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type SigningRecords []*SigningRecord

func (p SigningRecords) Sort() {
	sort.Slice(p, func(i, j int) bool {
		return p[i].GetId() < p[j].GetId()
	})
}

func (credential *CredentialStatus) Validate() error {
	if credential.GetDate() == 0 {
		return errors.New("empty signing credential date")
	}
	if credential.GetValidUntilDate() == 0 {
		return errors.New("empty signing record credential expiration date")
	}
	if credential.GetCertificatePem() == "" {
		return errors.New("empty signing record credential certificate")
	}
	serial := big.Int{}
	if _, ok := serial.SetString(credential.GetSerial(), 10); !ok {
		return errors.New("invalid signing record credential certificate serial number")
	}
	if _, err := uuid.Parse(credential.GetIssuerId()); err != nil {
		return fmt.Errorf("invalid signing record issuer's ID(%v): %w", credential.GetIssuerId(), err)
	}
	return nil
}

func (signingRecord *SigningRecord) Marshal() ([]byte, error) {
	return proto.Marshal(signingRecord)
}

func (signingRecord *SigningRecord) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, signingRecord)
}

func (signingRecord *SigningRecord) Validate() error {
	if signingRecord.GetId() == "" {
		return errors.New("empty signing record ID")
	}
	if _, err := uuid.Parse(signingRecord.GetId()); err != nil {
		return fmt.Errorf("invalid signing record ID(%v): %w", signingRecord.GetId(), err)
	}
	if signingRecord.GetDeviceId() != "" {
		if _, err := uuid.Parse(signingRecord.GetDeviceId()); err != nil {
			return fmt.Errorf("invalid signing record deviceID(%v): %w", signingRecord.GetDeviceId(), err)
		}
	}
	if signingRecord.GetCommonName() == "" {
		return errors.New("empty signing record commonName")
	}
	if signingRecord.GetOwner() == "" {
		return errors.New("empty signing record owner")
	}
	credential := signingRecord.GetCredential()
	if credential != nil {
		return credential.Validate()
	}
	return nil
}
