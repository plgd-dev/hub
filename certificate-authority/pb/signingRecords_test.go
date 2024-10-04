package pb_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/stretchr/testify/require"
)

func TestCredentialStatusValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   *pb.CredentialStatus
		wantErr bool
	}{
		{
			name: "Valid credential",
			input: &pb.CredentialStatus{
				Date:           1659462400000000000,
				ValidUntilDate: 1669462400000000000,
				CertificatePem: "valid-cert",
				Serial:         "1234567890",
				IssuerId:       uuid.New().String(),
			},
			wantErr: false,
		},
		{
			name: "Missing signing credential date",
			input: &pb.CredentialStatus{
				Date:           0,
				ValidUntilDate: 1669462400000000000,
				CertificatePem: "valid-cert",
				Serial:         "1234567890",
				IssuerId:       uuid.New().String(),
			},
			wantErr: true,
		},
		{
			name: "Missing signing credential expiration date",
			input: &pb.CredentialStatus{
				Date:           1659462400000000000,
				ValidUntilDate: 0,
				CertificatePem: "valid-cert",
				Serial:         "1234567890",
				IssuerId:       uuid.New().String(),
			},
			wantErr: true,
		},
		{
			name: "Missing signing record credential certificate",
			input: &pb.CredentialStatus{
				Date:           1659462400000000000,
				ValidUntilDate: 1669462400000000000,
				CertificatePem: "",
				Serial:         "1234567890",
				IssuerId:       uuid.New().String(),
			},
			wantErr: true,
		},
		{
			name: "Invalid certificate serial number",
			input: &pb.CredentialStatus{
				Date:           1659462400000000000,
				ValidUntilDate: 1669462400000000000,
				CertificatePem: "valid-cert",
				Serial:         "invalid-serial",
				IssuerId:       uuid.New().String(),
			},
			wantErr: true,
		},
		{
			name: "Invalid issuer ID",
			input: &pb.CredentialStatus{
				Date:           1659462400000000000,
				ValidUntilDate: 1669462400000000000,
				CertificatePem: "valid-cert",
				Serial:         "1234567890",
				IssuerId:       "invalid-uuid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSigningRecordValidate(t *testing.T) {
	validCredential := &pb.CredentialStatus{
		Date:           1659462400000000000,
		ValidUntilDate: 1669462400000000000,
		CertificatePem: "valid-cert",
		Serial:         "1234567890",
		IssuerId:       uuid.New().String(),
	}

	tests := []struct {
		name    string
		input   *pb.SigningRecord
		wantErr bool
	}{
		{
			name: "Valid signing record",
			input: &pb.SigningRecord{
				Id:         uuid.New().String(),
				Owner:      "owner",
				CommonName: "common_name",
				DeviceId:   uuid.New().String(),
				Credential: validCredential,
			},
			wantErr: false,
		},
		{
			name: "Missing signing record ID",
			input: &pb.SigningRecord{
				Id:         "",
				Owner:      "owner",
				CommonName: "common_name",
				DeviceId:   uuid.New().String(),
				Credential: validCredential,
			},
			wantErr: true,
		},
		{
			name: "Invalid signing record ID",
			input: &pb.SigningRecord{
				Id:         "invalid-uuid",
				Owner:      "owner",
				CommonName: "common_name",
				DeviceId:   uuid.New().String(),
				Credential: validCredential,
			},
			wantErr: true,
		},
		{
			name: "Invalid device ID",
			input: &pb.SigningRecord{
				Id:         uuid.New().String(),
				Owner:      "owner",
				CommonName: "common_name",
				DeviceId:   "invalid-uuid",
				Credential: validCredential,
			},
			wantErr: true,
		},
		{
			name: "Missing common name",
			input: &pb.SigningRecord{
				Id:         uuid.New().String(),
				Owner:      "owner",
				CommonName: "",
				DeviceId:   uuid.New().String(),
				Credential: validCredential,
			},
			wantErr: true,
		},
		{
			name: "Missing owner",
			input: &pb.SigningRecord{
				Id:         uuid.New().String(),
				Owner:      "",
				CommonName: "common_name",
				DeviceId:   uuid.New().String(),
				Credential: validCredential,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
