// TODO overit ze pending command sa nezmaze na neexistujucom resource ak sa device pripoji a nepublishne hned resource
// Overit correlation id - ak sa pouziva rovnake napriec viacerymi resourcami

// scenare
// - Uzivatel vie vytvorit config, automaticka (backend) inkrementacia verzie
// - Uzivatel updatne config, verzia sa inkrementuje, Modal -> chces aplikovat na vsetky uz provisionnute devici? Informovat uzivatela, ze niektore devici mozu byt offline a command moze vyexpirovat.
// - Uzivatel updatne config, verzia sa inkrementuje, informujeme uzivatela ze vsetky pending commandy z predoslej verzie budu cancelnute ako aj dalsie sekvencne updaty resourcov pre predoslu verziu

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v5.26.1
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
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

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
	GetConditions(ctx context.Context, in *GetConditionsRequest, opts ...grpc.CallOption) (SnippetService_GetConditionsClient, error)
	DeleteConditions(ctx context.Context, in *DeleteConditionsRequest, opts ...grpc.CallOption) (*DeleteConditionsResponse, error)
	UpdateCondition(ctx context.Context, in *Condition, opts ...grpc.CallOption) (*Condition, error)
	CreateConfiguration(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*Configuration, error)
	GetConfigurations(ctx context.Context, in *GetConfigurationsRequest, opts ...grpc.CallOption) (SnippetService_GetConfigurationsClient, error)
	DeleteConfigurations(ctx context.Context, in *DeleteConfigurationsRequest, opts ...grpc.CallOption) (*DeleteConfigurationsResponse, error)
	UpdateConfiguration(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*Configuration, error)
	// streaming process of update configuration to invoker
	InvokeConfiguration(ctx context.Context, in *InvokeConfigurationRequest, opts ...grpc.CallOption) (SnippetService_InvokeConfigurationClient, error)
	GetAppliedConfigurations(ctx context.Context, in *GetAppliedDeviceConfigurationsRequest, opts ...grpc.CallOption) (SnippetService_GetAppliedConfigurationsClient, error)
	DeleteAppliedConfigurations(ctx context.Context, in *DeleteAppliedDeviceConfigurationsRequest, opts ...grpc.CallOption) (*DeleteAppliedDeviceConfigurationsResponse, error)
}

type snippetServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewSnippetServiceClient(cc grpc.ClientConnInterface) SnippetServiceClient {
	return &snippetServiceClient{cc}
}

func (c *snippetServiceClient) CreateCondition(ctx context.Context, in *Condition, opts ...grpc.CallOption) (*Condition, error) {
	out := new(Condition)
	err := c.cc.Invoke(ctx, SnippetService_CreateCondition_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) GetConditions(ctx context.Context, in *GetConditionsRequest, opts ...grpc.CallOption) (SnippetService_GetConditionsClient, error) {
	stream, err := c.cc.NewStream(ctx, &SnippetService_ServiceDesc.Streams[0], SnippetService_GetConditions_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &snippetServiceGetConditionsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type SnippetService_GetConditionsClient interface {
	Recv() (*Condition, error)
	grpc.ClientStream
}

type snippetServiceGetConditionsClient struct {
	grpc.ClientStream
}

func (x *snippetServiceGetConditionsClient) Recv() (*Condition, error) {
	m := new(Condition)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *snippetServiceClient) DeleteConditions(ctx context.Context, in *DeleteConditionsRequest, opts ...grpc.CallOption) (*DeleteConditionsResponse, error) {
	out := new(DeleteConditionsResponse)
	err := c.cc.Invoke(ctx, SnippetService_DeleteConditions_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) UpdateCondition(ctx context.Context, in *Condition, opts ...grpc.CallOption) (*Condition, error) {
	out := new(Condition)
	err := c.cc.Invoke(ctx, SnippetService_UpdateCondition_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) CreateConfiguration(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*Configuration, error) {
	out := new(Configuration)
	err := c.cc.Invoke(ctx, SnippetService_CreateConfiguration_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) GetConfigurations(ctx context.Context, in *GetConfigurationsRequest, opts ...grpc.CallOption) (SnippetService_GetConfigurationsClient, error) {
	stream, err := c.cc.NewStream(ctx, &SnippetService_ServiceDesc.Streams[1], SnippetService_GetConfigurations_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &snippetServiceGetConfigurationsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type SnippetService_GetConfigurationsClient interface {
	Recv() (*Configuration, error)
	grpc.ClientStream
}

type snippetServiceGetConfigurationsClient struct {
	grpc.ClientStream
}

func (x *snippetServiceGetConfigurationsClient) Recv() (*Configuration, error) {
	m := new(Configuration)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *snippetServiceClient) DeleteConfigurations(ctx context.Context, in *DeleteConfigurationsRequest, opts ...grpc.CallOption) (*DeleteConfigurationsResponse, error) {
	out := new(DeleteConfigurationsResponse)
	err := c.cc.Invoke(ctx, SnippetService_DeleteConfigurations_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) UpdateConfiguration(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*Configuration, error) {
	out := new(Configuration)
	err := c.cc.Invoke(ctx, SnippetService_UpdateConfiguration_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *snippetServiceClient) InvokeConfiguration(ctx context.Context, in *InvokeConfigurationRequest, opts ...grpc.CallOption) (SnippetService_InvokeConfigurationClient, error) {
	stream, err := c.cc.NewStream(ctx, &SnippetService_ServiceDesc.Streams[2], SnippetService_InvokeConfiguration_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &snippetServiceInvokeConfigurationClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type SnippetService_InvokeConfigurationClient interface {
	Recv() (*AppliedDeviceConfiguration, error)
	grpc.ClientStream
}

type snippetServiceInvokeConfigurationClient struct {
	grpc.ClientStream
}

func (x *snippetServiceInvokeConfigurationClient) Recv() (*AppliedDeviceConfiguration, error) {
	m := new(AppliedDeviceConfiguration)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *snippetServiceClient) GetAppliedConfigurations(ctx context.Context, in *GetAppliedDeviceConfigurationsRequest, opts ...grpc.CallOption) (SnippetService_GetAppliedConfigurationsClient, error) {
	stream, err := c.cc.NewStream(ctx, &SnippetService_ServiceDesc.Streams[3], SnippetService_GetAppliedConfigurations_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &snippetServiceGetAppliedConfigurationsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type SnippetService_GetAppliedConfigurationsClient interface {
	Recv() (*AppliedDeviceConfiguration, error)
	grpc.ClientStream
}

type snippetServiceGetAppliedConfigurationsClient struct {
	grpc.ClientStream
}

func (x *snippetServiceGetAppliedConfigurationsClient) Recv() (*AppliedDeviceConfiguration, error) {
	m := new(AppliedDeviceConfiguration)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *snippetServiceClient) DeleteAppliedConfigurations(ctx context.Context, in *DeleteAppliedDeviceConfigurationsRequest, opts ...grpc.CallOption) (*DeleteAppliedDeviceConfigurationsResponse, error) {
	out := new(DeleteAppliedDeviceConfigurationsResponse)
	err := c.cc.Invoke(ctx, SnippetService_DeleteAppliedConfigurations_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SnippetServiceServer is the server API for SnippetService service.
// All implementations must embed UnimplementedSnippetServiceServer
// for forward compatibility
type SnippetServiceServer interface {
	CreateCondition(context.Context, *Condition) (*Condition, error)
	GetConditions(*GetConditionsRequest, SnippetService_GetConditionsServer) error
	DeleteConditions(context.Context, *DeleteConditionsRequest) (*DeleteConditionsResponse, error)
	UpdateCondition(context.Context, *Condition) (*Condition, error)
	CreateConfiguration(context.Context, *Configuration) (*Configuration, error)
	GetConfigurations(*GetConfigurationsRequest, SnippetService_GetConfigurationsServer) error
	DeleteConfigurations(context.Context, *DeleteConfigurationsRequest) (*DeleteConfigurationsResponse, error)
	UpdateConfiguration(context.Context, *Configuration) (*Configuration, error)
	// streaming process of update configuration to invoker
	InvokeConfiguration(*InvokeConfigurationRequest, SnippetService_InvokeConfigurationServer) error
	GetAppliedConfigurations(*GetAppliedDeviceConfigurationsRequest, SnippetService_GetAppliedConfigurationsServer) error
	DeleteAppliedConfigurations(context.Context, *DeleteAppliedDeviceConfigurationsRequest) (*DeleteAppliedDeviceConfigurationsResponse, error)
	mustEmbedUnimplementedSnippetServiceServer()
}

// UnimplementedSnippetServiceServer must be embedded to have forward compatible implementations.
type UnimplementedSnippetServiceServer struct {
}

func (UnimplementedSnippetServiceServer) CreateCondition(context.Context, *Condition) (*Condition, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateCondition not implemented")
}
func (UnimplementedSnippetServiceServer) GetConditions(*GetConditionsRequest, SnippetService_GetConditionsServer) error {
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
func (UnimplementedSnippetServiceServer) GetConfigurations(*GetConfigurationsRequest, SnippetService_GetConfigurationsServer) error {
	return status.Errorf(codes.Unimplemented, "method GetConfigurations not implemented")
}
func (UnimplementedSnippetServiceServer) DeleteConfigurations(context.Context, *DeleteConfigurationsRequest) (*DeleteConfigurationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteConfigurations not implemented")
}
func (UnimplementedSnippetServiceServer) UpdateConfiguration(context.Context, *Configuration) (*Configuration, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateConfiguration not implemented")
}
func (UnimplementedSnippetServiceServer) InvokeConfiguration(*InvokeConfigurationRequest, SnippetService_InvokeConfigurationServer) error {
	return status.Errorf(codes.Unimplemented, "method InvokeConfiguration not implemented")
}
func (UnimplementedSnippetServiceServer) GetAppliedConfigurations(*GetAppliedDeviceConfigurationsRequest, SnippetService_GetAppliedConfigurationsServer) error {
	return status.Errorf(codes.Unimplemented, "method GetAppliedConfigurations not implemented")
}
func (UnimplementedSnippetServiceServer) DeleteAppliedConfigurations(context.Context, *DeleteAppliedDeviceConfigurationsRequest) (*DeleteAppliedDeviceConfigurationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteAppliedConfigurations not implemented")
}
func (UnimplementedSnippetServiceServer) mustEmbedUnimplementedSnippetServiceServer() {}

// UnsafeSnippetServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SnippetServiceServer will
// result in compilation errors.
type UnsafeSnippetServiceServer interface {
	mustEmbedUnimplementedSnippetServiceServer()
}

func RegisterSnippetServiceServer(s grpc.ServiceRegistrar, srv SnippetServiceServer) {
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
	return srv.(SnippetServiceServer).GetConditions(m, &snippetServiceGetConditionsServer{stream})
}

type SnippetService_GetConditionsServer interface {
	Send(*Condition) error
	grpc.ServerStream
}

type snippetServiceGetConditionsServer struct {
	grpc.ServerStream
}

func (x *snippetServiceGetConditionsServer) Send(m *Condition) error {
	return x.ServerStream.SendMsg(m)
}

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
	return srv.(SnippetServiceServer).GetConfigurations(m, &snippetServiceGetConfigurationsServer{stream})
}

type SnippetService_GetConfigurationsServer interface {
	Send(*Configuration) error
	grpc.ServerStream
}

type snippetServiceGetConfigurationsServer struct {
	grpc.ServerStream
}

func (x *snippetServiceGetConfigurationsServer) Send(m *Configuration) error {
	return x.ServerStream.SendMsg(m)
}

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

func _SnippetService_InvokeConfiguration_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(InvokeConfigurationRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(SnippetServiceServer).InvokeConfiguration(m, &snippetServiceInvokeConfigurationServer{stream})
}

type SnippetService_InvokeConfigurationServer interface {
	Send(*AppliedDeviceConfiguration) error
	grpc.ServerStream
}

type snippetServiceInvokeConfigurationServer struct {
	grpc.ServerStream
}

func (x *snippetServiceInvokeConfigurationServer) Send(m *AppliedDeviceConfiguration) error {
	return x.ServerStream.SendMsg(m)
}

func _SnippetService_GetAppliedConfigurations_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetAppliedDeviceConfigurationsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(SnippetServiceServer).GetAppliedConfigurations(m, &snippetServiceGetAppliedConfigurationsServer{stream})
}

type SnippetService_GetAppliedConfigurationsServer interface {
	Send(*AppliedDeviceConfiguration) error
	grpc.ServerStream
}

type snippetServiceGetAppliedConfigurationsServer struct {
	grpc.ServerStream
}

func (x *snippetServiceGetAppliedConfigurationsServer) Send(m *AppliedDeviceConfiguration) error {
	return x.ServerStream.SendMsg(m)
}

func _SnippetService_DeleteAppliedConfigurations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteAppliedDeviceConfigurationsRequest)
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
		return srv.(SnippetServiceServer).DeleteAppliedConfigurations(ctx, req.(*DeleteAppliedDeviceConfigurationsRequest))
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
			StreamName:    "InvokeConfiguration",
			Handler:       _SnippetService_InvokeConfiguration_Handler,
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
