// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.3
// source: snippet-service/pb/service.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	SnippetService_CreateCondition_FullMethodName             = "/snippetservice.pb.SnippetService/CreateCondition"
	SnippetService_GetConditions_FullMethodName               = "/snippetservice.pb.SnippetService/GetConditions"
	SnippetService_DeleteConditions_FullMethodName            = "/snippetservice.pb.SnippetService/DeleteConditions"
	SnippetService_UpdateCondition_FullMethodName             = "/snippetservice.pb.SnippetService/UpdateCondition"
	SnippetService_CreateConfiguration_FullMethodName         = "/snippetservice.pb.SnippetService/CreateConfiguration"
	SnippetService_GetConfigurations_FullMethodName           = "/snippetservice.pb.SnippetService/GetConfigurations"
	SnippetService_DeleteConfigurations_FullMethodName        = "/snippetservice.pb.SnippetService/DeleteConfigurations"
	SnippetService_UpdateConfiguration_FullMethodName         = "/snippetservice.pb.SnippetService/UpdateConfiguration"
	SnippetService_InvokeConfiguration_FullMethodName         = "/snippetservice.pb.SnippetService/InvokeConfiguration"
	SnippetService_GetAppliedConfigurations_FullMethodName    = "/snippetservice.pb.SnippetService/GetAppliedConfigurations"
	SnippetService_DeleteAppliedConfigurations_FullMethodName = "/snippetservice.pb.SnippetService/DeleteAppliedConfigurations"
)

// SnippetServiceClient is the client API for SnippetService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SnippetServiceClient interface {
	CreateCondition(ctx context.Context, in *Condition, opts ...grpc.CallOption) (*Condition, error)
	GetConditions(ctx context.Context, in *GetConditionsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Condition], error)
	DeleteConditions(ctx context.Context, in *DeleteConditionsRequest, opts ...grpc.CallOption) (*DeleteConditionsResponse, error)
	// For update the condition whole condition is required and the version must be incremented.
	UpdateCondition(ctx context.Context, in *Condition, opts ...grpc.CallOption) (*Condition, error)
	CreateConfiguration(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*Configuration, error)
	GetConfigurations(ctx context.Context, in *GetConfigurationsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Configuration], error)
	DeleteConfigurations(ctx context.Context, in *DeleteConfigurationsRequest, opts ...grpc.CallOption) (*DeleteConfigurationsResponse, error)
	// For update the configuration whole configuration is required and the version must be incremented.
	UpdateConfiguration(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*Configuration, error)
	// streaming process of update configuration to invoker
	InvokeConfiguration(ctx context.Context, in *InvokeConfigurationRequest, opts ...grpc.CallOption) (*InvokeConfigurationResponse, error)
	GetAppliedConfigurations(ctx context.Context, in *GetAppliedConfigurationsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[AppliedConfiguration], error)
	DeleteAppliedConfigurations(ctx context.Context, in *DeleteAppliedConfigurationsRequest, opts ...grpc.CallOption) (*DeleteAppliedConfigurationsResponse, error)
}

type snippetServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewSnippetServiceClient(cc grpc.ClientConnInterface) SnippetServiceClient {
	return &snippetServiceClient{cc}
}

func (c *snippetServiceClient) CreateCondition(ctx context.Context, in *Condition, opts ...grpc.CallOption) (*Condition, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Condition)
	err := c.cc.Invoke(ctx, SnippetService_CreateCondition_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) GetConditions(ctx context.Context, in *GetConditionsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Condition], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &SnippetService_ServiceDesc.Streams[0], SnippetService_GetConditions_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[GetConditionsRequest, Condition]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type SnippetService_GetConditionsClient = grpc.ServerStreamingClient[Condition]

func (c *snippetServiceClient) DeleteConditions(ctx context.Context, in *DeleteConditionsRequest, opts ...grpc.CallOption) (*DeleteConditionsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteConditionsResponse)
	err := c.cc.Invoke(ctx, SnippetService_DeleteConditions_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) UpdateCondition(ctx context.Context, in *Condition, opts ...grpc.CallOption) (*Condition, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Condition)
	err := c.cc.Invoke(ctx, SnippetService_UpdateCondition_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) CreateConfiguration(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*Configuration, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Configuration)
	err := c.cc.Invoke(ctx, SnippetService_CreateConfiguration_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) GetConfigurations(ctx context.Context, in *GetConfigurationsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Configuration], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &SnippetService_ServiceDesc.Streams[1], SnippetService_GetConfigurations_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[GetConfigurationsRequest, Configuration]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type SnippetService_GetConfigurationsClient = grpc.ServerStreamingClient[Configuration]

func (c *snippetServiceClient) DeleteConfigurations(ctx context.Context, in *DeleteConfigurationsRequest, opts ...grpc.CallOption) (*DeleteConfigurationsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteConfigurationsResponse)
	err := c.cc.Invoke(ctx, SnippetService_DeleteConfigurations_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) UpdateConfiguration(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*Configuration, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Configuration)
	err := c.cc.Invoke(ctx, SnippetService_UpdateConfiguration_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) InvokeConfiguration(ctx context.Context, in *InvokeConfigurationRequest, opts ...grpc.CallOption) (*InvokeConfigurationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(InvokeConfigurationResponse)
	err := c.cc.Invoke(ctx, SnippetService_InvokeConfiguration_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) GetAppliedConfigurations(ctx context.Context, in *GetAppliedConfigurationsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[AppliedConfiguration], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &SnippetService_ServiceDesc.Streams[2], SnippetService_GetAppliedConfigurations_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[GetAppliedConfigurationsRequest, AppliedConfiguration]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type SnippetService_GetAppliedConfigurationsClient = grpc.ServerStreamingClient[AppliedConfiguration]

func (c *snippetServiceClient) DeleteAppliedConfigurations(ctx context.Context, in *DeleteAppliedConfigurationsRequest, opts ...grpc.CallOption) (*DeleteAppliedConfigurationsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteAppliedConfigurationsResponse)
	err := c.cc.Invoke(ctx, SnippetService_DeleteAppliedConfigurations_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SnippetServiceServer is the server API for SnippetService service.
// All implementations must embed UnimplementedSnippetServiceServer
// for forward compatibility.
type SnippetServiceServer interface {
	CreateCondition(context.Context, *Condition) (*Condition, error)
	GetConditions(*GetConditionsRequest, grpc.ServerStreamingServer[Condition]) error
	DeleteConditions(context.Context, *DeleteConditionsRequest) (*DeleteConditionsResponse, error)
	// For update the condition whole condition is required and the version must be incremented.
	UpdateCondition(context.Context, *Condition) (*Condition, error)
	CreateConfiguration(context.Context, *Configuration) (*Configuration, error)
	GetConfigurations(*GetConfigurationsRequest, grpc.ServerStreamingServer[Configuration]) error
	DeleteConfigurations(context.Context, *DeleteConfigurationsRequest) (*DeleteConfigurationsResponse, error)
	// For update the configuration whole configuration is required and the version must be incremented.
	UpdateConfiguration(context.Context, *Configuration) (*Configuration, error)
	// streaming process of update configuration to invoker
	InvokeConfiguration(context.Context, *InvokeConfigurationRequest) (*InvokeConfigurationResponse, error)
	GetAppliedConfigurations(*GetAppliedConfigurationsRequest, grpc.ServerStreamingServer[AppliedConfiguration]) error
	DeleteAppliedConfigurations(context.Context, *DeleteAppliedConfigurationsRequest) (*DeleteAppliedConfigurationsResponse, error)
	mustEmbedUnimplementedSnippetServiceServer()
}

// UnimplementedSnippetServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedSnippetServiceServer struct{}

func (UnimplementedSnippetServiceServer) CreateCondition(context.Context, *Condition) (*Condition, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateCondition not implemented")
}
func (UnimplementedSnippetServiceServer) GetConditions(*GetConditionsRequest, grpc.ServerStreamingServer[Condition]) error {
	return status.Errorf(codes.Unimplemented, "method GetConditions not implemented")
}
func (UnimplementedSnippetServiceServer) DeleteConditions(context.Context, *DeleteConditionsRequest) (*DeleteConditionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteConditions not implemented")
}
func (UnimplementedSnippetServiceServer) UpdateCondition(context.Context, *Condition) (*Condition, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateCondition not implemented")
}
func (UnimplementedSnippetServiceServer) CreateConfiguration(context.Context, *Configuration) (*Configuration, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateConfiguration not implemented")
}
func (UnimplementedSnippetServiceServer) GetConfigurations(*GetConfigurationsRequest, grpc.ServerStreamingServer[Configuration]) error {
	return status.Errorf(codes.Unimplemented, "method GetConfigurations not implemented")
}
func (UnimplementedSnippetServiceServer) DeleteConfigurations(context.Context, *DeleteConfigurationsRequest) (*DeleteConfigurationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteConfigurations not implemented")
}
func (UnimplementedSnippetServiceServer) UpdateConfiguration(context.Context, *Configuration) (*Configuration, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateConfiguration not implemented")
}
func (UnimplementedSnippetServiceServer) InvokeConfiguration(context.Context, *InvokeConfigurationRequest) (*InvokeConfigurationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InvokeConfiguration not implemented")
}
func (UnimplementedSnippetServiceServer) GetAppliedConfigurations(*GetAppliedConfigurationsRequest, grpc.ServerStreamingServer[AppliedConfiguration]) error {
	return status.Errorf(codes.Unimplemented, "method GetAppliedConfigurations not implemented")
}
func (UnimplementedSnippetServiceServer) DeleteAppliedConfigurations(context.Context, *DeleteAppliedConfigurationsRequest) (*DeleteAppliedConfigurationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteAppliedConfigurations not implemented")
}
func (UnimplementedSnippetServiceServer) mustEmbedUnimplementedSnippetServiceServer() {}
func (UnimplementedSnippetServiceServer) testEmbeddedByValue()                        {}

// UnsafeSnippetServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SnippetServiceServer will
// result in compilation errors.
type UnsafeSnippetServiceServer interface {
	mustEmbedUnimplementedSnippetServiceServer()
}

func RegisterSnippetServiceServer(s grpc.ServiceRegistrar, srv SnippetServiceServer) {
	// If the following call pancis, it indicates UnimplementedSnippetServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&SnippetService_ServiceDesc, srv)
}

func _SnippetService_CreateCondition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Condition)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SnippetServiceServer).CreateCondition(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SnippetService_CreateCondition_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SnippetServiceServer).CreateCondition(ctx, req.(*Condition))
	}
	return interceptor(ctx, in, info, handler)
}

func _SnippetService_GetConditions_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetConditionsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(SnippetServiceServer).GetConditions(m, &grpc.GenericServerStream[GetConditionsRequest, Condition]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type SnippetService_GetConditionsServer = grpc.ServerStreamingServer[Condition]

func _SnippetService_DeleteConditions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteConditionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SnippetServiceServer).DeleteConditions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SnippetService_DeleteConditions_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SnippetServiceServer).DeleteConditions(ctx, req.(*DeleteConditionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SnippetService_UpdateCondition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Condition)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SnippetServiceServer).UpdateCondition(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SnippetService_UpdateCondition_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SnippetServiceServer).UpdateCondition(ctx, req.(*Condition))
	}
	return interceptor(ctx, in, info, handler)
}

func _SnippetService_CreateConfiguration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Configuration)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SnippetServiceServer).CreateConfiguration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SnippetService_CreateConfiguration_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SnippetServiceServer).CreateConfiguration(ctx, req.(*Configuration))
	}
	return interceptor(ctx, in, info, handler)
}

func _SnippetService_GetConfigurations_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetConfigurationsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(SnippetServiceServer).GetConfigurations(m, &grpc.GenericServerStream[GetConfigurationsRequest, Configuration]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type SnippetService_GetConfigurationsServer = grpc.ServerStreamingServer[Configuration]

func _SnippetService_DeleteConfigurations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteConfigurationsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SnippetServiceServer).DeleteConfigurations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SnippetService_DeleteConfigurations_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SnippetServiceServer).DeleteConfigurations(ctx, req.(*DeleteConfigurationsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SnippetService_UpdateConfiguration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Configuration)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SnippetServiceServer).UpdateConfiguration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SnippetService_UpdateConfiguration_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SnippetServiceServer).UpdateConfiguration(ctx, req.(*Configuration))
	}
	return interceptor(ctx, in, info, handler)
}

func _SnippetService_InvokeConfiguration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InvokeConfigurationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SnippetServiceServer).InvokeConfiguration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SnippetService_InvokeConfiguration_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SnippetServiceServer).InvokeConfiguration(ctx, req.(*InvokeConfigurationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SnippetService_GetAppliedConfigurations_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetAppliedConfigurationsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(SnippetServiceServer).GetAppliedConfigurations(m, &grpc.GenericServerStream[GetAppliedConfigurationsRequest, AppliedConfiguration]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type SnippetService_GetAppliedConfigurationsServer = grpc.ServerStreamingServer[AppliedConfiguration]

func _SnippetService_DeleteAppliedConfigurations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteAppliedConfigurationsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SnippetServiceServer).DeleteAppliedConfigurations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SnippetService_DeleteAppliedConfigurations_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SnippetServiceServer).DeleteAppliedConfigurations(ctx, req.(*DeleteAppliedConfigurationsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// SnippetService_ServiceDesc is the grpc.ServiceDesc for SnippetService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SnippetService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "snippetservice.pb.SnippetService",
	HandlerType: (*SnippetServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateCondition",
			Handler:    _SnippetService_CreateCondition_Handler,
		},
		{
			MethodName: "DeleteConditions",
			Handler:    _SnippetService_DeleteConditions_Handler,
		},
		{
			MethodName: "UpdateCondition",
			Handler:    _SnippetService_UpdateCondition_Handler,
		},
		{
			MethodName: "CreateConfiguration",
			Handler:    _SnippetService_CreateConfiguration_Handler,
		},
		{
			MethodName: "DeleteConfigurations",
			Handler:    _SnippetService_DeleteConfigurations_Handler,
		},
		{
			MethodName: "UpdateConfiguration",
			Handler:    _SnippetService_UpdateConfiguration_Handler,
		},
		{
			MethodName: "InvokeConfiguration",
			Handler:    _SnippetService_InvokeConfiguration_Handler,
		},
		{
			MethodName: "DeleteAppliedConfigurations",
			Handler:    _SnippetService_DeleteAppliedConfigurations_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetConditions",
			Handler:       _SnippetService_GetConditions_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetConfigurations",
			Handler:       _SnippetService_GetConfigurations_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetAppliedConfigurations",
			Handler:       _SnippetService_GetAppliedConfigurations_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "snippet-service/pb/service.proto",
}
