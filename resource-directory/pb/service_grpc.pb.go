// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v5.26.1
// source: resource-directory/pb/service.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	ResourceDirectory_GetLatestDeviceETags_FullMethodName = "/resourcedirectory.pb.ResourceDirectory/GetLatestDeviceETags"
)

// ResourceDirectoryClient is the client API for ResourceDirectory service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ResourceDirectoryClient interface {
	// Get the most recent device etags, each corresponding to a different resource in order of most recent to least recent.
	GetLatestDeviceETags(ctx context.Context, in *GetLatestDeviceETagsRequest, opts ...grpc.CallOption) (*GetLatestDeviceETagsResponse, error)
}

type resourceDirectoryClient struct {
	cc grpc.ClientConnInterface
}

func NewResourceDirectoryClient(cc grpc.ClientConnInterface) ResourceDirectoryClient {
	return &resourceDirectoryClient{cc}
}

func (c *resourceDirectoryClient) GetLatestDeviceETags(ctx context.Context, in *GetLatestDeviceETagsRequest, opts ...grpc.CallOption) (*GetLatestDeviceETagsResponse, error) {
	out := new(GetLatestDeviceETagsResponse)
	err := c.cc.Invoke(ctx, ResourceDirectory_GetLatestDeviceETags_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ResourceDirectoryServer is the server API for ResourceDirectory service.
// All implementations must embed UnimplementedResourceDirectoryServer
// for forward compatibility
type ResourceDirectoryServer interface {
	// Get the most recent device etags, each corresponding to a different resource in order of most recent to least recent.
	GetLatestDeviceETags(context.Context, *GetLatestDeviceETagsRequest) (*GetLatestDeviceETagsResponse, error)
	mustEmbedUnimplementedResourceDirectoryServer()
}

// UnimplementedResourceDirectoryServer must be embedded to have forward compatible implementations.
type UnimplementedResourceDirectoryServer struct {
}

func (UnimplementedResourceDirectoryServer) GetLatestDeviceETags(context.Context, *GetLatestDeviceETagsRequest) (*GetLatestDeviceETagsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLatestDeviceETags not implemented")
}
func (UnimplementedResourceDirectoryServer) mustEmbedUnimplementedResourceDirectoryServer() {}

// UnsafeResourceDirectoryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ResourceDirectoryServer will
// result in compilation errors.
type UnsafeResourceDirectoryServer interface {
	mustEmbedUnimplementedResourceDirectoryServer()
}

func RegisterResourceDirectoryServer(s grpc.ServiceRegistrar, srv ResourceDirectoryServer) {
	s.RegisterService(&ResourceDirectory_ServiceDesc, srv)
}

func _ResourceDirectory_GetLatestDeviceETags_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetLatestDeviceETagsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceDirectoryServer).GetLatestDeviceETags(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ResourceDirectory_GetLatestDeviceETags_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceDirectoryServer).GetLatestDeviceETags(ctx, req.(*GetLatestDeviceETagsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ResourceDirectory_ServiceDesc is the grpc.ServiceDesc for ResourceDirectory service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ResourceDirectory_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "resourcedirectory.pb.ResourceDirectory",
	HandlerType: (*ResourceDirectoryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetLatestDeviceETags",
			Handler:    _ResourceDirectory_GetLatestDeviceETags_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "resource-directory/pb/service.proto",
}
