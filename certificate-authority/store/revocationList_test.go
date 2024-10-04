package store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/stretchr/testify/require"
)

func TestRevocationListCertificateValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   store.RevocationListCertificate
		wantErr bool
	}{
		{
			name: "Valid certificate",
			input: store.RevocationListCertificate{
				Serial:     "12345",
				Revocation: time.Now().UnixNano(),
			},
			wantErr: false,
		},
		{
			name: "Missing serial number",
			input: store.RevocationListCertificate{
				Serial:     "",
				Revocation: time.Now().UnixNano(),
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
	validCertificate := &store.RevocationListCertificate{
		Serial:     "12345",
		Revocation: time.Now().UnixNano(),
	}
	invalidCertificate := &store.RevocationListCertificate{
		Serial:     "",
		Revocation: time.Now().UnixNano(),
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
				IssuedAt:     time.Now().UnixNano(),
				ValidUntil:   time.Now().Add(time.Minute).UnixNano(),
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
				IssuedAt:     time.Now().UnixNano(),
				ValidUntil:   time.Now().Add(time.Minute).UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{validCertificate},
			},
			wantErr: true,
		},
		{
			name: "Missing issuedAt time",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				ValidUntil:   time.Now().Add(time.Minute).UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{validCertificate},
			},
			wantErr: true,
		},
		{
			name: "Missing validUntil time",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				IssuedAt:     time.Now().UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{validCertificate},
			},
			wantErr: true,
		},
		{
			name: "Invalid certificate in the list",
			input: store.RevocationList{
				Id:           uuid.New().String(),
				IssuedAt:     time.Now().UnixNano(),
				ValidUntil:   time.Now().Add(time.Minute).UnixNano(),
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{invalidCertificate},
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
