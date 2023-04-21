package store

import (
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
)

const (
	CommonNameKey     = "commonName"     // must match with pb.SigningRecord.CommonName tag
	OwnerKey          = "owner"          // must match with pb.SigningRecord.Owner tag
	DateKey           = "date"           // must match with pb.SigningRecord.Date tag
	CredentialKey     = "credential"     // must match with pb.SigningRecord.Credential tag
	CreationDateKey   = "creationDate"   // must match with pb.SigningRecord.CreationDate tag
	PublicKeyKey      = "publicKey"      // must match with pb.SigningRecord.PublicKey tag
	ValidUntilDateKey = "validUntilDate" // must match with pb.SigningRecord.Credential.ValidUntilDate tag
	DeviceIDKey       = "deviceId"       // must match with pb.SigningRecord.Credential.DeviceID tag
)

type SigningRecord = pb.SigningRecord
