// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: resource-aggregate/pb/resources.proto

package commands

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

type Status int32

const (
	Status_UNKNOWN            Status = 0
	Status_OK                 Status = 1
	Status_BAD_REQUEST        Status = 2
	Status_UNAUTHORIZED       Status = 3
	Status_FORBIDDEN          Status = 4
	Status_NOT_FOUND          Status = 5
	Status_UNAVAILABLE        Status = 6
	Status_NOT_IMPLEMENTED    Status = 7
	Status_ACCEPTED           Status = 8
	Status_ERROR              Status = 9
	Status_METHOD_NOT_ALLOWED Status = 10
	Status_CREATED            Status = 11
	Status_CANCELED           Status = 12 // Canceled indicates the operation was canceled (typically by the user).
)

// Enum value maps for Status.
var (
	Status_name = map[int32]string{
		0:  "UNKNOWN",
		1:  "OK",
		2:  "BAD_REQUEST",
		3:  "UNAUTHORIZED",
		4:  "FORBIDDEN",
		5:  "NOT_FOUND",
		6:  "UNAVAILABLE",
		7:  "NOT_IMPLEMENTED",
		8:  "ACCEPTED",
		9:  "ERROR",
		10: "METHOD_NOT_ALLOWED",
		11: "CREATED",
		12: "CANCELED",
	}
	Status_value = map[string]int32{
		"UNKNOWN":            0,
		"OK":                 1,
		"BAD_REQUEST":        2,
		"UNAUTHORIZED":       3,
		"FORBIDDEN":          4,
		"NOT_FOUND":          5,
		"UNAVAILABLE":        6,
		"NOT_IMPLEMENTED":    7,
		"ACCEPTED":           8,
		"ERROR":              9,
		"METHOD_NOT_ALLOWED": 10,
		"CREATED":            11,
		"CANCELED":           12,
	}
)

func (x Status) Enum() *Status {
	p := new(Status)
	*p = x
	return p
}

func (x Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Status) Descriptor() protoreflect.EnumDescriptor {
	return file_resource_aggregate_pb_resources_proto_enumTypes[0].Descriptor()
}

func (Status) Type() protoreflect.EnumType {
	return &file_resource_aggregate_pb_resources_proto_enumTypes[0]
}

func (x Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Status.Descriptor instead.
func (Status) EnumDescriptor() ([]byte, []int) {
	return file_resource_aggregate_pb_resources_proto_rawDescGZIP(), []int{0}
}

// https://github.com/openconnectivityfoundation/core/blob/master/schemas/oic.links.properties.core-schema.json
type Resource struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Href                  string                 `protobuf:"bytes,1,opt,name=href,proto3" json:"href,omitempty"`
	DeviceId              string                 `protobuf:"bytes,2,opt,name=device_id,json=deviceId,proto3" json:"device_id,omitempty"`
	ResourceTypes         []string               `protobuf:"bytes,3,rep,name=resource_types,json=resourceTypes,proto3" json:"resource_types,omitempty"`
	Interfaces            []string               `protobuf:"bytes,4,rep,name=interfaces,proto3" json:"interfaces,omitempty"`
	Anchor                string                 `protobuf:"bytes,5,opt,name=anchor,proto3" json:"anchor,omitempty"`
	Title                 string                 `protobuf:"bytes,6,opt,name=title,proto3" json:"title,omitempty"`
	SupportedContentTypes []string               `protobuf:"bytes,7,rep,name=supported_content_types,json=supportedContentTypes,proto3" json:"supported_content_types,omitempty"`
	ValidUntil            int64                  `protobuf:"varint,8,opt,name=valid_until,json=validUntil,proto3" json:"valid_until,omitempty"`
	Policy                *Policy                `protobuf:"bytes,9,opt,name=policy,proto3" json:"policy,omitempty"`
	EndpointInformations  []*EndpointInformation `protobuf:"bytes,10,rep,name=endpoint_informations,json=endpointInformations,proto3" json:"endpoint_informations,omitempty"`
}

func (x *Resource) Reset() {
	*x = Resource{}
	if protoimpl.UnsafeEnabled {
		mi := &file_resource_aggregate_pb_resources_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Resource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Resource) ProtoMessage() {}

func (x *Resource) ProtoReflect() protoreflect.Message {
	mi := &file_resource_aggregate_pb_resources_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Resource.ProtoReflect.Descriptor instead.
func (*Resource) Descriptor() ([]byte, []int) {
	return file_resource_aggregate_pb_resources_proto_rawDescGZIP(), []int{0}
}

func (x *Resource) GetHref() string {
	if x != nil {
		return x.Href
	}
	return ""
}

func (x *Resource) GetDeviceId() string {
	if x != nil {
		return x.DeviceId
	}
	return ""
}

func (x *Resource) GetResourceTypes() []string {
	if x != nil {
		return x.ResourceTypes
	}
	return nil
}

func (x *Resource) GetInterfaces() []string {
	if x != nil {
		return x.Interfaces
	}
	return nil
}

func (x *Resource) GetAnchor() string {
	if x != nil {
		return x.Anchor
	}
	return ""
}

func (x *Resource) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

func (x *Resource) GetSupportedContentTypes() []string {
	if x != nil {
		return x.SupportedContentTypes
	}
	return nil
}

func (x *Resource) GetValidUntil() int64 {
	if x != nil {
		return x.ValidUntil
	}
	return 0
}

func (x *Resource) GetPolicy() *Policy {
	if x != nil {
		return x.Policy
	}
	return nil
}

func (x *Resource) GetEndpointInformations() []*EndpointInformation {
	if x != nil {
		return x.EndpointInformations
	}
	return nil
}

type Policy struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BitFlags int32 `protobuf:"varint,1,opt,name=bit_flags,json=bitFlags,proto3" json:"bit_flags,omitempty"`
}

func (x *Policy) Reset() {
	*x = Policy{}
	if protoimpl.UnsafeEnabled {
		mi := &file_resource_aggregate_pb_resources_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Policy) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Policy) ProtoMessage() {}

func (x *Policy) ProtoReflect() protoreflect.Message {
	mi := &file_resource_aggregate_pb_resources_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Policy.ProtoReflect.Descriptor instead.
func (*Policy) Descriptor() ([]byte, []int) {
	return file_resource_aggregate_pb_resources_proto_rawDescGZIP(), []int{1}
}

func (x *Policy) GetBitFlags() int32 {
	if x != nil {
		return x.BitFlags
	}
	return 0
}

type Content struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data              []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	ContentType       string `protobuf:"bytes,2,opt,name=content_type,json=contentType,proto3" json:"content_type,omitempty"`
	CoapContentFormat int32  `protobuf:"varint,3,opt,name=coap_content_format,json=coapContentFormat,proto3" json:"coap_content_format,omitempty"` // -1 means content-format was not provided
}

func (x *Content) Reset() {
	*x = Content{}
	if protoimpl.UnsafeEnabled {
		mi := &file_resource_aggregate_pb_resources_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Content) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Content) ProtoMessage() {}

func (x *Content) ProtoReflect() protoreflect.Message {
	mi := &file_resource_aggregate_pb_resources_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Content.ProtoReflect.Descriptor instead.
func (*Content) Descriptor() ([]byte, []int) {
	return file_resource_aggregate_pb_resources_proto_rawDescGZIP(), []int{2}
}

func (x *Content) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *Content) GetContentType() string {
	if x != nil {
		return x.ContentType
	}
	return ""
}

func (x *Content) GetCoapContentFormat() int32 {
	if x != nil {
		return x.CoapContentFormat
	}
	return 0
}

type EndpointInformation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Endpoint string `protobuf:"bytes,1,opt,name=endpoint,proto3" json:"endpoint,omitempty"`
	Priority int64  `protobuf:"varint,2,opt,name=priority,proto3" json:"priority,omitempty"`
}

func (x *EndpointInformation) Reset() {
	*x = EndpointInformation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_resource_aggregate_pb_resources_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EndpointInformation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EndpointInformation) ProtoMessage() {}

func (x *EndpointInformation) ProtoReflect() protoreflect.Message {
	mi := &file_resource_aggregate_pb_resources_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EndpointInformation.ProtoReflect.Descriptor instead.
func (*EndpointInformation) Descriptor() ([]byte, []int) {
	return file_resource_aggregate_pb_resources_proto_rawDescGZIP(), []int{3}
}

func (x *EndpointInformation) GetEndpoint() string {
	if x != nil {
		return x.Endpoint
	}
	return ""
}

func (x *EndpointInformation) GetPriority() int64 {
	if x != nil {
		return x.Priority
	}
	return 0
}

var File_resource_aggregate_pb_resources_proto protoreflect.FileDescriptor

var file_resource_aggregate_pb_resources_proto_rawDesc = []byte{
	0x0a, 0x25, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2d, 0x61, 0x67, 0x67, 0x72, 0x65,
	0x67, 0x61, 0x74, 0x65, 0x2f, 0x70, 0x62, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x14, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x62, 0x22, 0x9f, 0x03,
	0x0a, 0x08, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x72,
	0x65, 0x66, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x72, 0x65, 0x66, 0x12, 0x1b,
	0x0a, 0x09, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x49, 0x64, 0x12, 0x25, 0x0a, 0x0e, 0x72,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x18, 0x03, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x0d, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x54, 0x79, 0x70,
	0x65, 0x73, 0x12, 0x1e, 0x0a, 0x0a, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x73,
	0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x6e, 0x63, 0x68, 0x6f, 0x72, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x06, 0x61, 0x6e, 0x63, 0x68, 0x6f, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x69,
	0x74, 0x6c, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x69, 0x74, 0x6c, 0x65,
	0x12, 0x36, 0x0a, 0x17, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x64, 0x5f, 0x63, 0x6f,
	0x6e, 0x74, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x15, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x64, 0x43, 0x6f, 0x6e, 0x74,
	0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x73, 0x12, 0x1f, 0x0a, 0x0b, 0x76, 0x61, 0x6c, 0x69,
	0x64, 0x5f, 0x75, 0x6e, 0x74, 0x69, 0x6c, 0x18, 0x08, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0a, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x55, 0x6e, 0x74, 0x69, 0x6c, 0x12, 0x34, 0x0a, 0x06, 0x70, 0x6f, 0x6c,
	0x69, 0x63, 0x79, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x62,
	0x2e, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x52, 0x06, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x12,
	0x5e, 0x0a, 0x15, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x5f, 0x69, 0x6e, 0x66, 0x6f,
	0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x0a, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x29,
	0x2e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61,
	0x74, 0x65, 0x2e, 0x70, 0x62, 0x2e, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x49, 0x6e,
	0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x14, 0x65, 0x6e, 0x64, 0x70, 0x6f,
	0x69, 0x6e, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22,
	0x25, 0x0a, 0x06, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x12, 0x1b, 0x0a, 0x09, 0x62, 0x69, 0x74,
	0x5f, 0x66, 0x6c, 0x61, 0x67, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x62, 0x69,
	0x74, 0x46, 0x6c, 0x61, 0x67, 0x73, 0x22, 0x70, 0x0a, 0x07, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e,
	0x74, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x6f, 0x6e,
	0x74, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x12, 0x2e, 0x0a, 0x13, 0x63, 0x6f, 0x61, 0x70,
	0x5f, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x5f, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x11, 0x63, 0x6f, 0x61, 0x70, 0x43, 0x6f, 0x6e, 0x74, 0x65,
	0x6e, 0x74, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x22, 0x4d, 0x0a, 0x13, 0x45, 0x6e, 0x64, 0x70,
	0x6f, 0x69, 0x6e, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12,
	0x1a, 0x0a, 0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x70,
	0x72, 0x69, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x70,
	0x72, 0x69, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x2a, 0xd0, 0x01, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12,
	0x06, 0x0a, 0x02, 0x4f, 0x4b, 0x10, 0x01, 0x12, 0x0f, 0x0a, 0x0b, 0x42, 0x41, 0x44, 0x5f, 0x52,
	0x45, 0x51, 0x55, 0x45, 0x53, 0x54, 0x10, 0x02, 0x12, 0x10, 0x0a, 0x0c, 0x55, 0x4e, 0x41, 0x55,
	0x54, 0x48, 0x4f, 0x52, 0x49, 0x5a, 0x45, 0x44, 0x10, 0x03, 0x12, 0x0d, 0x0a, 0x09, 0x46, 0x4f,
	0x52, 0x42, 0x49, 0x44, 0x44, 0x45, 0x4e, 0x10, 0x04, 0x12, 0x0d, 0x0a, 0x09, 0x4e, 0x4f, 0x54,
	0x5f, 0x46, 0x4f, 0x55, 0x4e, 0x44, 0x10, 0x05, 0x12, 0x0f, 0x0a, 0x0b, 0x55, 0x4e, 0x41, 0x56,
	0x41, 0x49, 0x4c, 0x41, 0x42, 0x4c, 0x45, 0x10, 0x06, 0x12, 0x13, 0x0a, 0x0f, 0x4e, 0x4f, 0x54,
	0x5f, 0x49, 0x4d, 0x50, 0x4c, 0x45, 0x4d, 0x45, 0x4e, 0x54, 0x45, 0x44, 0x10, 0x07, 0x12, 0x0c,
	0x0a, 0x08, 0x41, 0x43, 0x43, 0x45, 0x50, 0x54, 0x45, 0x44, 0x10, 0x08, 0x12, 0x09, 0x0a, 0x05,
	0x45, 0x52, 0x52, 0x4f, 0x52, 0x10, 0x09, 0x12, 0x16, 0x0a, 0x12, 0x4d, 0x45, 0x54, 0x48, 0x4f,
	0x44, 0x5f, 0x4e, 0x4f, 0x54, 0x5f, 0x41, 0x4c, 0x4c, 0x4f, 0x57, 0x45, 0x44, 0x10, 0x0a, 0x12,
	0x0b, 0x0a, 0x07, 0x43, 0x52, 0x45, 0x41, 0x54, 0x45, 0x44, 0x10, 0x0b, 0x12, 0x0c, 0x0a, 0x08,
	0x43, 0x41, 0x4e, 0x43, 0x45, 0x4c, 0x45, 0x44, 0x10, 0x0c, 0x42, 0x41, 0x5a, 0x3f, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x6c, 0x67, 0x64, 0x2d, 0x64, 0x65,
	0x76, 0x2f, 0x68, 0x75, 0x62, 0x2f, 0x76, 0x32, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x2d, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x2f, 0x63, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x73, 0x3b, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x73, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_resource_aggregate_pb_resources_proto_rawDescOnce sync.Once
	file_resource_aggregate_pb_resources_proto_rawDescData = file_resource_aggregate_pb_resources_proto_rawDesc
)

func file_resource_aggregate_pb_resources_proto_rawDescGZIP() []byte {
	file_resource_aggregate_pb_resources_proto_rawDescOnce.Do(func() {
		file_resource_aggregate_pb_resources_proto_rawDescData = protoimpl.X.CompressGZIP(file_resource_aggregate_pb_resources_proto_rawDescData)
	})
	return file_resource_aggregate_pb_resources_proto_rawDescData
}

var file_resource_aggregate_pb_resources_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_resource_aggregate_pb_resources_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_resource_aggregate_pb_resources_proto_goTypes = []interface{}{
	(Status)(0),                 // 0: resourceaggregate.pb.Status
	(*Resource)(nil),            // 1: resourceaggregate.pb.Resource
	(*Policy)(nil),              // 2: resourceaggregate.pb.Policy
	(*Content)(nil),             // 3: resourceaggregate.pb.Content
	(*EndpointInformation)(nil), // 4: resourceaggregate.pb.EndpointInformation
}
var file_resource_aggregate_pb_resources_proto_depIdxs = []int32{
	2, // 0: resourceaggregate.pb.Resource.policy:type_name -> resourceaggregate.pb.Policy
	4, // 1: resourceaggregate.pb.Resource.endpoint_informations:type_name -> resourceaggregate.pb.EndpointInformation
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_resource_aggregate_pb_resources_proto_init() }
func file_resource_aggregate_pb_resources_proto_init() {
	if File_resource_aggregate_pb_resources_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_resource_aggregate_pb_resources_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Resource); i {
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
		file_resource_aggregate_pb_resources_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Policy); i {
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
		file_resource_aggregate_pb_resources_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Content); i {
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
		file_resource_aggregate_pb_resources_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EndpointInformation); i {
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
			RawDescriptor: file_resource_aggregate_pb_resources_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_resource_aggregate_pb_resources_proto_goTypes,
		DependencyIndexes: file_resource_aggregate_pb_resources_proto_depIdxs,
		EnumInfos:         file_resource_aggregate_pb_resources_proto_enumTypes,
		MessageInfos:      file_resource_aggregate_pb_resources_proto_msgTypes,
	}.Build()
	File_resource_aggregate_pb_resources_proto = out.File
	file_resource_aggregate_pb_resources_proto_rawDesc = nil
	file_resource_aggregate_pb_resources_proto_goTypes = nil
	file_resource_aggregate_pb_resources_proto_depIdxs = nil
}