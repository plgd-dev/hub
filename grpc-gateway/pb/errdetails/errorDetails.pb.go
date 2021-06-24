// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.14.0
// source: github.com/plgd-dev/cloud/grpc-gateway/pb/errdetails/errorDetails.proto

package errdetails

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

type Content struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data        []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	ContentType string `protobuf:"bytes,2,opt,name=content_type,json=contentType,proto3" json:"content_type,omitempty"`
}

func (x *Content) Reset() {
	*x = Content{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Content) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Content) ProtoMessage() {}

func (x *Content) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_msgTypes[0]
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
	return file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDescGZIP(), []int{0}
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

type DeviceError struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Content *Content `protobuf:"bytes,2,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *DeviceError) Reset() {
	*x = DeviceError{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeviceError) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeviceError) ProtoMessage() {}

func (x *DeviceError) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeviceError.ProtoReflect.Descriptor instead.
func (*DeviceError) Descriptor() ([]byte, []int) {
	return file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDescGZIP(), []int{1}
}

func (x *DeviceError) GetContent() *Content {
	if x != nil {
		return x.Content
	}
	return nil
}

var File_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto protoreflect.FileDescriptor

var file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDesc = []byte{
	0x0a, 0x47, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x6c, 0x67,
	0x64, 0x2d, 0x64, 0x65, 0x76, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x67, 0x72, 0x70, 0x63,
	0x2d, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2f, 0x70, 0x62, 0x2f, 0x65, 0x72, 0x72, 0x64,
	0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x2f, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x44, 0x65, 0x74, 0x61,
	0x69, 0x6c, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x23, 0x6f, 0x63, 0x66, 0x2e, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79,
	0x2e, 0x70, 0x62, 0x2e, 0x65, 0x72, 0x72, 0x64, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x22, 0x40,
	0x0a, 0x07, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74,
	0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x21, 0x0a,
	0x0c, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65,
	0x22, 0x55, 0x0a, 0x0b, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x12,
	0x46, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x2c, 0x2e, 0x6f, 0x63, 0x66, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x67, 0x72, 0x70,
	0x63, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x65, 0x72, 0x72, 0x64,
	0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x52, 0x07,
	0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x42, 0x41, 0x5a, 0x3f, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x6c, 0x67, 0x64, 0x2d, 0x64, 0x65, 0x76, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2d, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61,
	0x79, 0x2f, 0x70, 0x62, 0x2f, 0x65, 0x72, 0x72, 0x64, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x3b,
	0x65, 0x72, 0x72, 0x64, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDescOnce sync.Once
	file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDescData = file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDesc
)

func file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDescGZIP() []byte {
	file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDescOnce.Do(func() {
		file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDescData)
	})
	return file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDescData
}

var file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_goTypes = []interface{}{
	(*Content)(nil),     // 0: ocf.cloud.grpcgateway.pb.errdetails.Content
	(*DeviceError)(nil), // 1: ocf.cloud.grpcgateway.pb.errdetails.DeviceError
}
var file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_depIdxs = []int32{
	0, // 0: ocf.cloud.grpcgateway.pb.errdetails.DeviceError.content:type_name -> ocf.cloud.grpcgateway.pb.errdetails.Content
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_init() }
func file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_init() {
	if File_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeviceError); i {
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
			RawDescriptor: file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_goTypes,
		DependencyIndexes: file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_depIdxs,
		MessageInfos:      file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_msgTypes,
	}.Build()
	File_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto = out.File
	file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_rawDesc = nil
	file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_goTypes = nil
	file_github_com_plgd_dev_cloud_grpc_gateway_pb_errdetails_errorDetails_proto_depIdxs = nil
}
