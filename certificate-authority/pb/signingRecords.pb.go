// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v5.27.3
// source: certificate-authority/pb/signingRecords.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetSigningRecordsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Filter by id.
	IdFilter []string `protobuf:"bytes,1,rep,name=id_filter,json=idFilter,proto3" json:"id_filter,omitempty"`
	// Filter by common_name.
	CommonNameFilter []string `protobuf:"bytes,2,rep,name=common_name_filter,json=commonNameFilter,proto3" json:"common_name_filter,omitempty"`
	// Filter by device_id - provides only identity certificates.
	DeviceIdFilter []string `protobuf:"bytes,3,rep,name=device_id_filter,json=deviceIdFilter,proto3" json:"device_id_filter,omitempty"`
}

func (x *GetSigningRecordsRequest) Reset() {
	*x = GetSigningRecordsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetSigningRecordsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetSigningRecordsRequest) ProtoMessage() {}

func (x *GetSigningRecordsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetSigningRecordsRequest.ProtoReflect.Descriptor instead.
func (*GetSigningRecordsRequest) Descriptor() ([]byte, []int) {
	return file_certificate_authority_pb_signingRecords_proto_rawDescGZIP(), []int{0}
}

func (x *GetSigningRecordsRequest) GetIdFilter() []string {
	if x != nil {
		return x.IdFilter
	}
	return nil
}

func (x *GetSigningRecordsRequest) GetCommonNameFilter() []string {
	if x != nil {
		return x.CommonNameFilter
	}
	return nil
}

func (x *GetSigningRecordsRequest) GetDeviceIdFilter() []string {
	if x != nil {
		return x.DeviceIdFilter
	}
	return nil
}

type CredentialStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Last time the device requested provisioning, in unix nanoseconds timestamp format.
	Date int64 `protobuf:"varint,1,opt,name=date,proto3" json:"date,omitempty" bson:"date"` // @gotags: bson:"date"
	// Last certificate issued.
	CertificatePem string `protobuf:"bytes,2,opt,name=certificate_pem,json=certificatePem,proto3" json:"certificate_pem,omitempty" bson:"identityCertificate"` // @gotags: bson:"identityCertificate"
	// Record valid until date, in unix nanoseconds timestamp format
	ValidUntilDate int64 `protobuf:"varint,3,opt,name=valid_until_date,json=validUntilDate,proto3" json:"valid_until_date,omitempty" bson:"validUntilDate"` // @gotags: bson:"validUntilDate"
	// Serial number of the last certificate issued
	Serial string `protobuf:"bytes,4,opt,name=serial,proto3" json:"serial,omitempty" bson:"serial"` // @gotags: bson:"serial"
	// Issuer id is calculated from the issuer's public certificate, and it is computed as uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw)
	IssuerId string `protobuf:"bytes,5,opt,name=issuer_id,json=issuerId,proto3" json:"issuer_id,omitempty" bson:"issuerId"` // @gotags: bson:"issuerId"
}

func (x *CredentialStatus) Reset() {
	*x = CredentialStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CredentialStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CredentialStatus) ProtoMessage() {}

func (x *CredentialStatus) ProtoReflect() protoreflect.Message {
	mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CredentialStatus.ProtoReflect.Descriptor instead.
func (*CredentialStatus) Descriptor() ([]byte, []int) {
	return file_certificate_authority_pb_signingRecords_proto_rawDescGZIP(), []int{1}
}

func (x *CredentialStatus) GetDate() int64 {
	if x != nil {
		return x.Date
	}
	return 0
}

func (x *CredentialStatus) GetCertificatePem() string {
	if x != nil {
		return x.CertificatePem
	}
	return ""
}

func (x *CredentialStatus) GetValidUntilDate() int64 {
	if x != nil {
		return x.ValidUntilDate
	}
	return 0
}

func (x *CredentialStatus) GetSerial() string {
	if x != nil {
		return x.Serial
	}
	return ""
}

func (x *CredentialStatus) GetIssuerId() string {
	if x != nil {
		return x.IssuerId
	}
	return ""
}

type SigningRecord struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The registration ID is determined by applying a formula that utilizes the certificate properties, and it is computed as uuid.NewSHA1(uuid.NameSpaceX500, common_name + uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw)).
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" bson:"_id"` // @gotags: bson:"_id"
	// Certificate owner.
	Owner string `protobuf:"bytes,2,opt,name=owner,proto3" json:"owner,omitempty" bson:"owner"` // @gotags: bson:"owner"
	// Common name of the certificate. If device_id is provided in the common name, then for update public key must be same.
	CommonName string `protobuf:"bytes,3,opt,name=common_name,json=commonName,proto3" json:"common_name,omitempty" bson:"commonName"` // @gotags: bson:"commonName"
	// DeviceID of the identity certificate.
	DeviceId string `protobuf:"bytes,4,opt,name=device_id,json=deviceId,proto3" json:"device_id,omitempty" bson:"deviceId,omitempty"` // @gotags: bson:"deviceId,omitempty"
	// Public key fingerprint in uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw) of the certificate.
	PublicKey string `protobuf:"bytes,5,opt,name=public_key,json=publicKey,proto3" json:"public_key,omitempty" bson:"publicKey"` // @gotags: bson:"publicKey"
	// Record creation date, in unix nanoseconds timestamp format
	CreationDate int64 `protobuf:"varint,6,opt,name=creation_date,json=creationDate,proto3" json:"creation_date,omitempty" bson:"creationDate,omitempty"` // @gotags: bson:"creationDate,omitempty"
	// Last credential provision overview.
	Credential *CredentialStatus `protobuf:"bytes,7,opt,name=credential,proto3" json:"credential,omitempty" bson:"credential"` // @gotags: bson:"credential"
}

func (x *SigningRecord) Reset() {
	*x = SigningRecord{}
	if protoimpl.UnsafeEnabled {
		mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SigningRecord) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SigningRecord) ProtoMessage() {}

func (x *SigningRecord) ProtoReflect() protoreflect.Message {
	mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SigningRecord.ProtoReflect.Descriptor instead.
func (*SigningRecord) Descriptor() ([]byte, []int) {
	return file_certificate_authority_pb_signingRecords_proto_rawDescGZIP(), []int{2}
}

func (x *SigningRecord) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *SigningRecord) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *SigningRecord) GetCommonName() string {
	if x != nil {
		return x.CommonName
	}
	return ""
}

func (x *SigningRecord) GetDeviceId() string {
	if x != nil {
		return x.DeviceId
	}
	return ""
}

func (x *SigningRecord) GetPublicKey() string {
	if x != nil {
		return x.PublicKey
	}
	return ""
}

func (x *SigningRecord) GetCreationDate() int64 {
	if x != nil {
		return x.CreationDate
	}
	return 0
}

func (x *SigningRecord) GetCredential() *CredentialStatus {
	if x != nil {
		return x.Credential
	}
	return nil
}

type DeleteSigningRecordsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Filter by id.
	IdFilter []string `protobuf:"bytes,1,rep,name=id_filter,json=idFilter,proto3" json:"id_filter,omitempty"`
	// Filter by common_name.
	DeviceIdFilter []string `protobuf:"bytes,2,rep,name=device_id_filter,json=deviceIdFilter,proto3" json:"device_id_filter,omitempty"`
}

func (x *DeleteSigningRecordsRequest) Reset() {
	*x = DeleteSigningRecordsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteSigningRecordsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteSigningRecordsRequest) ProtoMessage() {}

func (x *DeleteSigningRecordsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteSigningRecordsRequest.ProtoReflect.Descriptor instead.
func (*DeleteSigningRecordsRequest) Descriptor() ([]byte, []int) {
	return file_certificate_authority_pb_signingRecords_proto_rawDescGZIP(), []int{3}
}

func (x *DeleteSigningRecordsRequest) GetIdFilter() []string {
	if x != nil {
		return x.IdFilter
	}
	return nil
}

func (x *DeleteSigningRecordsRequest) GetDeviceIdFilter() []string {
	if x != nil {
		return x.DeviceIdFilter
	}
	return nil
}

// Revoke or delete certificates
type DeletedSigningRecords struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Number of deleted records.
	Count int64 `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
}

func (x *DeletedSigningRecords) Reset() {
	*x = DeletedSigningRecords{}
	if protoimpl.UnsafeEnabled {
		mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeletedSigningRecords) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeletedSigningRecords) ProtoMessage() {}

func (x *DeletedSigningRecords) ProtoReflect() protoreflect.Message {
	mi := &file_certificate_authority_pb_signingRecords_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeletedSigningRecords.ProtoReflect.Descriptor instead.
func (*DeletedSigningRecords) Descriptor() ([]byte, []int) {
	return file_certificate_authority_pb_signingRecords_proto_rawDescGZIP(), []int{4}
}

func (x *DeletedSigningRecords) GetCount() int64 {
	if x != nil {
		return x.Count
	}
	return 0
}

var File_certificate_authority_pb_signingRecords_proto protoreflect.FileDescriptor

var file_certificate_authority_pb_signingRecords_proto_rawDesc = []byte{
	0x0a, 0x2d, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x2d, 0x61, 0x75,
	0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x2f, 0x70, 0x62, 0x2f, 0x73, 0x69, 0x67, 0x6e, 0x69,
	0x6e, 0x67, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x17, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x61, 0x75, 0x74, 0x68,
	0x6f, 0x72, 0x69, 0x74, 0x79, 0x2e, 0x70, 0x62, 0x22, 0x8f, 0x01, 0x0a, 0x18, 0x47, 0x65, 0x74,
	0x53, 0x69, 0x67, 0x6e, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x73, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x69, 0x64, 0x5f, 0x66, 0x69, 0x6c, 0x74,
	0x65, 0x72, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x69, 0x64, 0x46, 0x69, 0x6c, 0x74,
	0x65, 0x72, 0x12, 0x2c, 0x0a, 0x12, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x5f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x10,
	0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x4e, 0x61, 0x6d, 0x65, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72,
	0x12, 0x28, 0x0a, 0x10, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x5f, 0x66, 0x69,
	0x6c, 0x74, 0x65, 0x72, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0e, 0x64, 0x65, 0x76, 0x69,
	0x63, 0x65, 0x49, 0x64, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x22, 0xae, 0x01, 0x0a, 0x10, 0x43,
	0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x64,
	0x61, 0x74, 0x65, 0x12, 0x27, 0x0a, 0x0f, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61,
	0x74, 0x65, 0x5f, 0x70, 0x65, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x63, 0x65,
	0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x50, 0x65, 0x6d, 0x12, 0x28, 0x0a, 0x10,
	0x76, 0x61, 0x6c, 0x69, 0x64, 0x5f, 0x75, 0x6e, 0x74, 0x69, 0x6c, 0x5f, 0x64, 0x61, 0x74, 0x65,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0e, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x55, 0x6e, 0x74,
	0x69, 0x6c, 0x44, 0x61, 0x74, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x65, 0x72, 0x69, 0x61, 0x6c,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x65, 0x72, 0x69, 0x61, 0x6c, 0x12, 0x1b,
	0x0a, 0x09, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x49, 0x64, 0x22, 0x82, 0x02, 0x0a, 0x0d,
	0x53, 0x69, 0x67, 0x6e, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x12, 0x0e, 0x0a,
	0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a,
	0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f, 0x77,
	0x6e, 0x65, 0x72, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e,
	0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69,
	0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x49,
	0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x6b, 0x65, 0x79, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79,
	0x12, 0x23, 0x0a, 0x0d, 0x63, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x64, 0x61, 0x74,
	0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c, 0x63, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x44, 0x61, 0x74, 0x65, 0x12, 0x49, 0x0a, 0x0a, 0x63, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74,
	0x69, 0x61, 0x6c, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x63, 0x65, 0x72, 0x74,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x79,
	0x2e, 0x70, 0x62, 0x2e, 0x43, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x52, 0x0a, 0x63, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c,
	0x22, 0x64, 0x0a, 0x1b, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x53, 0x69, 0x67, 0x6e, 0x69, 0x6e,
	0x67, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x1b, 0x0a, 0x09, 0x69, 0x64, 0x5f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x08, 0x69, 0x64, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12, 0x28, 0x0a, 0x10,
	0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x5f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72,
	0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0e, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x49, 0x64,
	0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x22, 0x2d, 0x0a, 0x15, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65,
	0x64, 0x53, 0x69, 0x67, 0x6e, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x73, 0x12,
	0x14, 0x0a, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x42, 0x38, 0x5a, 0x36, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x6c, 0x67, 0x64, 0x2d, 0x64, 0x65, 0x76, 0x2f, 0x68, 0x75, 0x62,
	0x2f, 0x76, 0x32, 0x2f, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x2d,
	0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x2f, 0x70, 0x62, 0x3b, 0x70, 0x62, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_certificate_authority_pb_signingRecords_proto_rawDescOnce sync.Once
	file_certificate_authority_pb_signingRecords_proto_rawDescData = file_certificate_authority_pb_signingRecords_proto_rawDesc
)

func file_certificate_authority_pb_signingRecords_proto_rawDescGZIP() []byte {
	file_certificate_authority_pb_signingRecords_proto_rawDescOnce.Do(func() {
		file_certificate_authority_pb_signingRecords_proto_rawDescData = protoimpl.X.CompressGZIP(file_certificate_authority_pb_signingRecords_proto_rawDescData)
	})
	return file_certificate_authority_pb_signingRecords_proto_rawDescData
}

var file_certificate_authority_pb_signingRecords_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_certificate_authority_pb_signingRecords_proto_goTypes = []any{
	(*GetSigningRecordsRequest)(nil),    // 0: certificateauthority.pb.GetSigningRecordsRequest
	(*CredentialStatus)(nil),            // 1: certificateauthority.pb.CredentialStatus
	(*SigningRecord)(nil),               // 2: certificateauthority.pb.SigningRecord
	(*DeleteSigningRecordsRequest)(nil), // 3: certificateauthority.pb.DeleteSigningRecordsRequest
	(*DeletedSigningRecords)(nil),       // 4: certificateauthority.pb.DeletedSigningRecords
}
var file_certificate_authority_pb_signingRecords_proto_depIdxs = []int32{
	1, // 0: certificateauthority.pb.SigningRecord.credential:type_name -> certificateauthority.pb.CredentialStatus
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_certificate_authority_pb_signingRecords_proto_init() }
func file_certificate_authority_pb_signingRecords_proto_init() {
	if File_certificate_authority_pb_signingRecords_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_certificate_authority_pb_signingRecords_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*GetSigningRecordsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_certificate_authority_pb_signingRecords_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*CredentialStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_certificate_authority_pb_signingRecords_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*SigningRecord); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_certificate_authority_pb_signingRecords_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*DeleteSigningRecordsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_certificate_authority_pb_signingRecords_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*DeletedSigningRecords); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_certificate_authority_pb_signingRecords_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_certificate_authority_pb_signingRecords_proto_goTypes,
		DependencyIndexes: file_certificate_authority_pb_signingRecords_proto_depIdxs,
		MessageInfos:      file_certificate_authority_pb_signingRecords_proto_msgTypes,
	}.Build()
	File_certificate_authority_pb_signingRecords_proto = out.File
	file_certificate_authority_pb_signingRecords_proto_rawDesc = nil
	file_certificate_authority_pb_signingRecords_proto_goTypes = nil
	file_certificate_authority_pb_signingRecords_proto_depIdxs = nil
}
