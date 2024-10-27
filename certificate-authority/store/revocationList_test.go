package store_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/stretchr/testify/require"
)

func TestParseBigInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *big.Int
		wantErr bool
	}{
		{
			name:  "Valid number",
			input: "123456789",
			want:  big.NewInt(123456789),
		},
		{
			name:    "Invalid Input - Non-numeric String",
			input:   "abcd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, err := store.ParseBigInt(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.Equal(t, tt.want, num)
		})
	}
}

func TestRevocationListCertificateValidate(t *testing.T) {
	revocation := time.Date(2042, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		input   store.RevocationListCertificate
		wantErr bool
	}{
		{
			name: "Valid certificate",
			input: store.RevocationListCertificate{
				Serial:     "12345",
				Revocation: revocation.UnixNano(),
			},
			wantErr: false,
		},
		{
			name: "Missing serial number",
			input: store.RevocationListCertificate{
				Serial:     "",
				Revocation: revocation.UnixNano(),
			},
			wantErr: true,
		},
		{
			name: "Missing revocation time",
			input: store.RevocationListCertificate{
				Serial:     "12345",
				Revocation: 0,
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

func TestRevocationListValidate(t *testing.T) {
	revocation := time.Date(2042, 1, 1, 0, 0, 0, 0, time.UTC)
	validCertificate := &store.RevocationListCertificate{
		Serial:     "12345",
		Revocation: revocation.UnixNano(),
	}
	invalidCertificate := &store.RevocationListCertificate{
		Serial:     "",
		Revocation: revocation.UnixNano(),
	}

	tests := []struct {
		name    string
		input   store.RevocationList
		wantErr bool
	}{
		{
			name: "Valid revocation list",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				IssuedAt:     revocation.UnixNano(),
				ValidUntil:   revocation.Add(time.Minute).UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{validCertificate},
			},
			wantErr: false,
		},
		{
			name: "Valid not-issued revocation list",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{validCertificate},
			},
			wantErr: false,
		},
		{
			name: "Invalid UUID",
			input: store.RevocationList{
				Id:           "invalid-uuid",
				IssuedAt:     revocation.UnixNano(),
				ValidUntil:   revocation.Add(time.Minute).UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{validCertificate},
			},
			wantErr: true,
		},
		{
			name: "Missing issuedAt time",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				ValidUntil:   revocation.Add(time.Minute).UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{validCertificate},
			},
			wantErr: true,
		},
		{
			name: "Missing validUntil time",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				IssuedAt:     revocation.UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{validCertificate},
			},
			wantErr: true,
		},
		{
			name: "ValidUntil before IssuedAt",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				IssuedAt:     revocation.Add(time.Hour).UnixNano(),
				ValidUntil:   revocation.UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{validCertificate},
			},
			wantErr: true,
		},
		{
			name: "Invalid certificate in the list",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				IssuedAt:     revocation.UnixNano(),
				ValidUntil:   revocation.Add(time.Minute).UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{invalidCertificate},
			},
			wantErr: true,
		},
		{
			name: "Invalid Number",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				IssuedAt:     revocation.UnixNano(),
				ValidUntil:   revocation.Add(time.Minute).UnixNano(),
				Number:       "not-a-number",
				Certificates: []*store.RevocationListCertificate{validCertificate},
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
