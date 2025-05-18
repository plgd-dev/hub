package pb

import (
	"sort"

	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/credential"
)

type ProvisioningRecords []*ProvisioningRecord

func (p ProvisioningRecords) Sort() {
	sort.Slice(p, func(i, j int) bool {
		return p[i].GetId() < p[j].GetId()
	})
}

func AccessControlSubjectToDevicePb(subject acl.Subject) *AccessControlDeviceSubject {
	if subject.Subject_Device == nil {
		return nil
	}
	return &AccessControlDeviceSubject{
		DeviceId: subject.DeviceID,
	}
}

func AccessControlSubjectToRolePb(subject acl.Subject) *AccessControlRoleSubject {
	if subject.Subject_Role == nil {
		return nil
	}
	return &AccessControlRoleSubject{
		Role:      subject.Role,
		Authority: subject.Authority,
	}
}

func AccessControlSubjectToConnectionPb(subject acl.Subject) *AccessControlConnectionSubject {
	if subject.Subject_Connection == nil {
		return nil
	}
	var v AccessControlConnectionSubject_ConnectionType
	switch subject.Type {
	case acl.ConnectionType_ANON_CLEAR:
		v = AccessControlConnectionSubject_ANON_CLEAR
	case acl.ConnectionType_AUTH_CRYPT:
		v = AccessControlConnectionSubject_AUTH_CRYPT
	}
	return &AccessControlConnectionSubject{
		Type: v,
	}
}

func AccessControlPermissionToPb(permission acl.Permission) []AccessControl_Permission {
	var res []AccessControl_Permission
	if permission.Has(acl.Permission_CREATE) {
		res = append(res, AccessControl_CREATE)
	}
	if permission.Has(acl.Permission_READ) {
		res = append(res, AccessControl_READ)
	}
	if permission.Has(acl.Permission_WRITE) {
		res = append(res, AccessControl_WRITE)
	}
	if permission.Has(acl.Permission_DELETE) {
		res = append(res, AccessControl_DELETE)
	}
	if permission.Has(acl.Permission_NOTIFY) {
		res = append(res, AccessControl_NOTIFY)
	}
	return res
}

func AccessControlResourcesToPb(resources []acl.Resource) []*AccessControlResource {
	res := make([]*AccessControlResource, 0, len(resources))
	for _, r := range resources {
		wildcard := AccessControlResource_NONE
		switch r.Wildcard {
		case acl.ResourceWildcard_NONCFG_SEC_ENDPOINT:
			wildcard = AccessControlResource_NONCFG_SEC_ENDPOINT
		case acl.ResourceWildcard_NONCFG_NONSEC_ENDPOINT:
			wildcard = AccessControlResource_NONCFG_NONSEC_ENDPOINT
		case acl.ResourceWildcard_NONCFG_ALL:
			wildcard = AccessControlResource_NONCFG_ALL
		}
		res = append(res, &AccessControlResource{
			Href:          r.Href,
			Interfaces:    r.Interfaces,
			ResourceTypes: r.ResourceTypes,
			Wildcard:      wildcard,
		})
	}
	return res
}

func DeviceAccessControlToPb(ac acl.AccessControl) *AccessControl {
	return &AccessControl{
		DeviceSubject:     AccessControlSubjectToDevicePb(ac.Subject),
		RoleSubject:       AccessControlSubjectToRolePb(ac.Subject),
		ConnectionSubject: AccessControlSubjectToConnectionPb(ac.Subject),
		Permissions:       AccessControlPermissionToPb(ac.Permission),
		Resources:         AccessControlResourcesToPb(ac.Resources),
	}
}

func DeviceAccessControlListToPb(acls []acl.AccessControl) []*AccessControl {
	acl := make([]*AccessControl, 0, len(acls))
	for _, ac := range acls {
		acl = append(acl, DeviceAccessControlToPb(ac))
	}
	return acl
}

var credentialTypes = map[credential.CredentialType]Credential_CredentialType{
	credential.CredentialType_SYMMETRIC_PAIR_WISE:                 Credential_SYMMETRIC_PAIR_WISE,
	credential.CredentialType_SYMMETRIC_GROUP:                     Credential_SYMMETRIC_GROUP,
	credential.CredentialType_ASYMMETRIC_SIGNING:                  Credential_ASYMMETRIC_SIGNING,
	credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE: Credential_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
	credential.CredentialType_PIN_OR_PASSWORD:                     Credential_PIN_OR_PASSWORD,
	credential.CredentialType_ASYMMETRIC_ENCRYPTION_KEY:           Credential_ASYMMETRIC_ENCRYPTION_KEY,
}

func CredentialTypeToPb(t credential.CredentialType) []Credential_CredentialType {
	var types []Credential_CredentialType
	if t == credential.CredentialType_EMPTY {
		return types
	}
	for key, ct := range credentialTypes {
		if t.Has(key) {
			types = append(types, ct)
		}
	}
	return types
}

var credentialUsages = map[credential.CredentialUsage]Credential_CredentialUsage{
	credential.CredentialUsage_CERT:         Credential_CERT,
	credential.CredentialUsage_MFG_CERT:     Credential_MFG_CERT,
	credential.CredentialUsage_MFG_TRUST_CA: Credential_MFG_TRUST_CA,
	credential.CredentialUsage_TRUST_CA:     Credential_TRUST_CA,
	credential.CredentialUsage_ROLE_CERT:    Credential_ROLE_CERT,
}

func CredentialUsageToPb(u credential.CredentialUsage) Credential_CredentialUsage {
	v, ok := credentialUsages[u]
	if !ok {
		return Credential_NONE
	}
	return v
}

func CredentialRoleIDToPb(roleID *credential.CredentialRoleID) *CredentialRoleID {
	if roleID == nil {
		return nil
	}
	return &CredentialRoleID{
		Authority: roleID.Authority,
		Role:      roleID.Role,
	}
}

var credentialRefreshMethods = map[credential.CredentialRefreshMethod]Credential_CredentialRefreshMethod{
	credential.CredentialRefreshMethod_KEY_AGREEMENT_PROTOCOL:                Credential_KEY_AGREEMENT_PROTOCOL,
	credential.CredentialRefreshMethod_KEY_AGREEMENT_PROTOCOL_AND_RANDOM_PIN: Credential_KEY_AGREEMENT_PROTOCOL_AND_RANDOM_PIN,
	credential.CredentialRefreshMethod_PROVISION_SERVICE:                     Credential_PROVISION_SERVICE,
	credential.CredentialRefreshMethod_KEY_DISTRIBUTION_SERVICE:              Credential_KEY_DISTRIBUTION_SERVICE,
	credential.CredentialRefreshMethod_PKCS10_REQUEST_TO_CA:                  Credential_PKCS10_REQUEST_TO_CA,
}

func CredentialSupportedRefreshMethodsToPb(methods []credential.CredentialRefreshMethod) []Credential_CredentialRefreshMethod {
	var res []Credential_CredentialRefreshMethod
	for _, m := range methods {
		if v, ok := credentialRefreshMethods[m]; ok {
			res = append(res, v)
		} else {
			res = append(res, Credential_UNKNOWN)
		}
	}
	return res
}

var credentialOptionalDataEncodings = map[credential.CredentialOptionalDataEncoding]CredentialOptionalData_Encoding{
	credential.CredentialOptionalDataEncoding_BASE64: CredentialOptionalData_BASE64,
	credential.CredentialOptionalDataEncoding_RAW:    CredentialOptionalData_RAW,
	credential.CredentialOptionalDataEncoding_CWT:    CredentialOptionalData_CWT,
	credential.CredentialOptionalDataEncoding_JWT:    CredentialOptionalData_JWT,
	credential.CredentialOptionalDataEncoding_DER:    CredentialOptionalData_DER,
	credential.CredentialOptionalDataEncoding_PEM:    CredentialOptionalData_PEM,
}

func CredentialOptionalDataEncodingToPb(encoding credential.CredentialOptionalDataEncoding) CredentialOptionalData_Encoding {
	v, ok := credentialOptionalDataEncodings[encoding]
	if !ok {
		return CredentialOptionalData_UNKNOWN
	}
	return v
}

func CredentialOptionalDataToPb(data *credential.CredentialOptionalData) *CredentialOptionalData {
	if data == nil {
		return nil
	}
	return &CredentialOptionalData{
		Data:      data.Data(),
		Encoding:  CredentialOptionalDataEncodingToPb(data.Encoding),
		IsRevoked: data.IsRevoked,
	}
}

var credentialPrivateDataEncodings = map[credential.CredentialPrivateDataEncoding]CredentialPrivateData_Encoding{
	credential.CredentialPrivateDataEncoding_BASE64: CredentialPrivateData_BASE64,
	credential.CredentialPrivateDataEncoding_RAW:    CredentialPrivateData_RAW,
	credential.CredentialPrivateDataEncoding_CWT:    CredentialPrivateData_CWT,
	credential.CredentialPrivateDataEncoding_JWT:    CredentialPrivateData_JWT,
	credential.CredentialPrivateDataEncoding_URI:    CredentialPrivateData_URI,
	credential.CredentialPrivateDataEncoding_HANDLE: CredentialPrivateData_HANDLE,
}

func CredentialPrivateDataEncodingToPb(encoding credential.CredentialPrivateDataEncoding) CredentialPrivateData_Encoding {
	v, ok := credentialPrivateDataEncodings[encoding]
	if !ok {
		return CredentialPrivateData_UNKNOWN
	}
	return v
}

func CredentialPrivateDataToPb(data *credential.CredentialPrivateData) *CredentialPrivateData {
	if data == nil {
		return nil
	}
	return &CredentialPrivateData{
		Data:     data.Data(),
		Encoding: CredentialPrivateDataEncodingToPb(data.Encoding),
		Handle:   int64(data.Handle),
	}
}

var credentialPublicDataEncodings = map[credential.CredentialPublicDataEncoding]CredentialPublicData_Encoding{
	credential.CredentialPublicDataEncoding_BASE64: CredentialPublicData_BASE64,
	credential.CredentialPublicDataEncoding_RAW:    CredentialPublicData_RAW,
	credential.CredentialPublicDataEncoding_CWT:    CredentialPublicData_CWT,
	credential.CredentialPublicDataEncoding_JWT:    CredentialPublicData_JWT,
	credential.CredentialPublicDataEncoding_URI:    CredentialPublicData_URI,
	credential.CredentialPublicDataEncoding_DER:    CredentialPublicData_DER,
	credential.CredentialPublicDataEncoding_PEM:    CredentialPublicData_PEM,
}

func CredentialPublicDataEncodingToPb(encoding credential.CredentialPublicDataEncoding) CredentialPublicData_Encoding {
	v, ok := credentialPublicDataEncodings[encoding]
	if !ok {
		return CredentialPublicData_UNKNOWN
	}
	return v
}

func CredentialPublicDataToPb(data *credential.CredentialPublicData) *CredentialPublicData {
	if data == nil {
		return nil
	}
	return &CredentialPublicData{
		Data:     data.Data(),
		Encoding: CredentialPublicDataEncodingToPb(data.Encoding),
	}
}

func CredentialToPb(cred credential.Credential) *Credential {
	return &Credential{
		Id:                      int64(cred.ID),
		Type:                    CredentialTypeToPb(cred.Type),
		Usage:                   CredentialUsageToPb(cred.Usage),
		Subject:                 cred.Subject,
		Period:                  cred.Period,
		RoleId:                  CredentialRoleIDToPb(cred.RoleID),
		SupportedRefreshMethods: CredentialSupportedRefreshMethodsToPb(cred.SupportedRefreshMethods),
		OptionalData:            CredentialOptionalDataToPb(cred.OptionalData),
		PrivateData:             CredentialPrivateDataToPb(cred.PrivateData),
		PublicData:              CredentialPublicDataToPb(cred.PublicData),
	}
}

func CredentialsToPb(creds []credential.Credential) []*Credential {
	pbCreds := make([]*Credential, 0, len(creds))
	for _, c := range creds {
		pbCreds = append(pbCreds, CredentialToPb(c))
	}
	return pbCreds
}
