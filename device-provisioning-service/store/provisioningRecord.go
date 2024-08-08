package store

import (
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
)

const (
	EnrollmentGroupIDKey = "enrollmentGroupId" // must match with pb.ProvisioningRecord.EnrollmentGroupID tag
	DeviceIDKey          = "deviceId"          // must match with pb.ProvisioningRecord.DeviceID tag
	AttestationKey       = "attestation"       // must match with pb.ProvisioningRecord.Attestation tag
	DateKey              = "date"              // must match with pb.ProvisioningRecord.Date tag
	CloudKey             = "cloud"             // must match with pb.ProvisioningRecord.Cloud tag
	ACLKey               = "acl"               // must match with pb.ProvisioningRecord.Acl tag
	CredentialKey        = "credential"        // must match with pb.ProvisioningRecord.Credential tag
	PlgdTimeKey          = "plgdTime"          // must match with pb.ProvisioningRecord.PlgdTime tag
	OwnershipKey         = "ownership"         // must match with pb.ProvisioningRecord.Ownership tag
	StatusKey            = "status"            // must match with pb.ProvisioningRecord.Status tag
	CreationDateKey      = "creationDate"      // must match with pb.ProvisioningRecord.CreationDate tag
	LocalEndpointsKey    = "localEndpoints"    // must match with pb.ProvisioningRecord.LocalEndpoints tag
	OwnerKey             = "owner"             // must match with pb.ProvisioningRecord.Owner tag
)

type ProvisioningRecord = pb.ProvisioningRecord
