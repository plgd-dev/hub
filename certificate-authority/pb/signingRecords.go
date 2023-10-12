package pb

import (
	"fmt"
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

func (signingRecord *SigningRecord) Marshal() ([]byte, error) {
	return proto.Marshal(signingRecord)
}

func (signingRecord *SigningRecord) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, signingRecord)
}

func (signingRecord *SigningRecord) Validate() error {
	if signingRecord.GetId() == "" {
		return fmt.Errorf("empty signing record ID")
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
		return fmt.Errorf("empty signing record commonName")
	}
	if signingRecord.GetOwner() == "" {
		return fmt.Errorf("empty signing record owner")
	}
	if signingRecord.GetCredential() != nil && signingRecord.GetCredential().GetDate() == 0 {
		return fmt.Errorf("empty signing credential date")
	}
	if signingRecord.GetCredential() != nil && signingRecord.GetCredential().GetValidUntilDate() == 0 {
		return fmt.Errorf("empty signing record credential expiration date")
	}
	if signingRecord.GetCredential() != nil && signingRecord.GetCredential().GetCertificatePem() == "" {
		return fmt.Errorf("empty signing record credential certificate")
	}
	return nil
}
