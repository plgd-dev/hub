package store

import (
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
)

const (
	IDKey                                          = "_id"                                           // must match with pb.ProvisioningRecord.Id, pb.EnrollmentGroup.Id tag
	HubIDKey                                       = "hubId"                                         // must match with pb.ProvisioningRecord.HubId
	HubIDsKey                                      = "hubIds"                                        // must match with pb.EnrollmentGroup.HubId
	AttestationMechanismX509LeadCertificateNameKey = "attestationMechanism.x509.leadCertificateName" // must match with all tags in path pb.EnrollmentGroup.AttestationMechanism.X509.LeadCertificateName
)

type EnrollmentGroup = pb.EnrollmentGroup
