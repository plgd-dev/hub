// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package pb

import (
	context "context"
	events "github.com/plgd-dev/cloud/resource-aggregate/events"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// GrpcGatewayClient is the client API for GrpcGateway service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GrpcGatewayClient interface {
	// Get all devices
	GetDevices(ctx context.Context, in *GetDevicesRequest, opts ...grpc.CallOption) (GrpcGateway_GetDevicesClient, error)
	// Get resource links of devices.
	GetResourceLinks(ctx context.Context, in *GetResourceLinksRequest, opts ...grpc.CallOption) (GrpcGateway_GetResourceLinksClient, error)
	// Get resource from the device.
	GetResourceFromDevice(ctx context.Context, in *GetResourceFromDeviceRequest, opts ...grpc.CallOption) (*GetResourceFromDeviceResponse, error)
	// Get resources from the resource shadow.
	GetResources(ctx context.Context, in *GetResourcesRequest, opts ...grpc.CallOption) (GrpcGateway_GetResourcesClient, error)
	// Update resource at the device.
	UpdateResource(ctx context.Context, in *UpdateResourceRequest, opts ...grpc.CallOption) (*UpdateResourceResponse, error)
	// Subscribe to events
	SubscribeToEvents(ctx context.Context, opts ...grpc.CallOption) (GrpcGateway_SubscribeToEventsClient, error)
	// Get client configuration
	GetClientConfiguration(ctx context.Context, in *ClientConfigurationRequest, opts ...grpc.CallOption) (*ClientConfigurationResponse, error)
	// Delete resource at the device.
	DeleteResource(ctx context.Context, in *DeleteResourceRequest, opts ...grpc.CallOption) (*DeleteResourceResponse, error)
	// Create resource at the device.
	CreateResource(ctx context.Context, in *CreateResourceRequest, opts ...grpc.CallOption) (*CreateResourceResponse, error)
	// Enables/disables shadow synchronization for device.
	UpdateDeviceMetadata(ctx context.Context, in *UpdateDeviceMetadataRequest, opts ...grpc.CallOption) (*UpdateDeviceMetadataResponse, error)
	// Gets pending commands for devices .
	GetPendingCommands(ctx context.Context, in *GetPendingCommandsRequest, opts ...grpc.CallOption) (GrpcGateway_GetPendingCommandsClient, error)
	// Cancels resource commands.
	CancelPendingCommands(ctx context.Context, in *CancelPendingCommandsRequest, opts ...grpc.CallOption) (*CancelPendingCommandsResponse, error)
	// Cancels device metadata updates.
	CancelPendingMetadataUpdates(ctx context.Context, in *CancelPendingMetadataUpdatesRequest, opts ...grpc.CallOption) (*CancelPendingCommandsResponse, error)
	// Gets metadata of the devices. Is contains online/offline or shadown synchronization status.
	GetDevicesMetadata(ctx context.Context, in *GetDevicesMetadataRequest, opts ...grpc.CallOption) (GrpcGateway_GetDevicesMetadataClient, error)
	// Get events for given combination of device id, resource id and timestamp
	GetEvents(ctx context.Context, in *GetEventsRequest, opts ...grpc.CallOption) (GrpcGateway_GetEventsClient, error)
}

type grpcGatewayClient struct {
	cc grpc.ClientConnInterface
}

func NewGrpcGatewayClient(cc grpc.ClientConnInterface) GrpcGatewayClient {
	return &grpcGatewayClient{cc}
}

func (c *grpcGatewayClient) GetDevices(ctx context.Context, in *GetDevicesRequest, opts ...grpc.CallOption) (GrpcGateway_GetDevicesClient, error) {
	stream, err := c.cc.NewStream(ctx, &GrpcGateway_ServiceDesc.Streams[0], "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetDevices", opts...)
	if err != nil {
		return nil, err
	}
	x := &grpcGatewayGetDevicesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GrpcGateway_GetDevicesClient interface {
	Recv() (*Device, error)
	grpc.ClientStream
}

type grpcGatewayGetDevicesClient struct {
	grpc.ClientStream
}

func (x *grpcGatewayGetDevicesClient) Recv() (*Device, error) {
	m := new(Device)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *grpcGatewayClient) GetResourceLinks(ctx context.Context, in *GetResourceLinksRequest, opts ...grpc.CallOption) (GrpcGateway_GetResourceLinksClient, error) {
	stream, err := c.cc.NewStream(ctx, &GrpcGateway_ServiceDesc.Streams[1], "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetResourceLinks", opts...)
	if err != nil {
		return nil, err
	}
	x := &grpcGatewayGetResourceLinksClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GrpcGateway_GetResourceLinksClient interface {
	Recv() (*events.ResourceLinksPublished, error)
	grpc.ClientStream
}

type grpcGatewayGetResourceLinksClient struct {
	grpc.ClientStream
}

func (x *grpcGatewayGetResourceLinksClient) Recv() (*events.ResourceLinksPublished, error) {
	m := new(events.ResourceLinksPublished)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *grpcGatewayClient) GetResourceFromDevice(ctx context.Context, in *GetResourceFromDeviceRequest, opts ...grpc.CallOption) (*GetResourceFromDeviceResponse, error) {
	out := new(GetResourceFromDeviceResponse)
	err := c.cc.Invoke(ctx, "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetResourceFromDevice", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *grpcGatewayClient) GetResources(ctx context.Context, in *GetResourcesRequest, opts ...grpc.CallOption) (GrpcGateway_GetResourcesClient, error) {
	stream, err := c.cc.NewStream(ctx, &GrpcGateway_ServiceDesc.Streams[2], "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetResources", opts...)
	if err != nil {
		return nil, err
	}
	x := &grpcGatewayGetResourcesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GrpcGateway_GetResourcesClient interface {
	Recv() (*Resource, error)
	grpc.ClientStream
}

type grpcGatewayGetResourcesClient struct {
	grpc.ClientStream
}

func (x *grpcGatewayGetResourcesClient) Recv() (*Resource, error) {
	m := new(Resource)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *grpcGatewayClient) UpdateResource(ctx context.Context, in *UpdateResourceRequest, opts ...grpc.CallOption) (*UpdateResourceResponse, error) {
	out := new(UpdateResourceResponse)
	err := c.cc.Invoke(ctx, "/ocf.cloud.grpcgateway.pb.GrpcGateway/UpdateResource", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *grpcGatewayClient) SubscribeToEvents(ctx context.Context, opts ...grpc.CallOption) (GrpcGateway_SubscribeToEventsClient, error) {
	stream, err := c.cc.NewStream(ctx, &GrpcGateway_ServiceDesc.Streams[3], "/ocf.cloud.grpcgateway.pb.GrpcGateway/SubscribeToEvents", opts...)
	if err != nil {
		return nil, err
	}
	x := &grpcGatewaySubscribeToEventsClient{stream}
	return x, nil
}

type GrpcGateway_SubscribeToEventsClient interface {
	Send(*SubscribeToEvents) error
	Recv() (*Event, error)
	grpc.ClientStream
}

type grpcGatewaySubscribeToEventsClient struct {
	grpc.ClientStream
}

func (x *grpcGatewaySubscribeToEventsClient) Send(m *SubscribeToEvents) error {
	return x.ClientStream.SendMsg(m)
}

func (x *grpcGatewaySubscribeToEventsClient) Recv() (*Event, error) {
	m := new(Event)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *grpcGatewayClient) GetClientConfiguration(ctx context.Context, in *ClientConfigurationRequest, opts ...grpc.CallOption) (*ClientConfigurationResponse, error) {
	out := new(ClientConfigurationResponse)
	err := c.cc.Invoke(ctx, "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetClientConfiguration", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *grpcGatewayClient) DeleteResource(ctx context.Context, in *DeleteResourceRequest, opts ...grpc.CallOption) (*DeleteResourceResponse, error) {
	out := new(DeleteResourceResponse)
	err := c.cc.Invoke(ctx, "/ocf.cloud.grpcgateway.pb.GrpcGateway/DeleteResource", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *grpcGatewayClient) CreateResource(ctx context.Context, in *CreateResourceRequest, opts ...grpc.CallOption) (*CreateResourceResponse, error) {
	out := new(CreateResourceResponse)
	err := c.cc.Invoke(ctx, "/ocf.cloud.grpcgateway.pb.GrpcGateway/CreateResource", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *grpcGatewayClient) UpdateDeviceMetadata(ctx context.Context, in *UpdateDeviceMetadataRequest, opts ...grpc.CallOption) (*UpdateDeviceMetadataResponse, error) {
	out := new(UpdateDeviceMetadataResponse)
	err := c.cc.Invoke(ctx, "/ocf.cloud.grpcgateway.pb.GrpcGateway/UpdateDeviceMetadata", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *grpcGatewayClient) GetPendingCommands(ctx context.Context, in *GetPendingCommandsRequest, opts ...grpc.CallOption) (GrpcGateway_GetPendingCommandsClient, error) {
	stream, err := c.cc.NewStream(ctx, &GrpcGateway_ServiceDesc.Streams[4], "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetPendingCommands", opts...)
	if err != nil {
		return nil, err
	}
	x := &grpcGatewayGetPendingCommandsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GrpcGateway_GetPendingCommandsClient interface {
	Recv() (*PendingCommand, error)
	grpc.ClientStream
}

type grpcGatewayGetPendingCommandsClient struct {
	grpc.ClientStream
}

func (x *grpcGatewayGetPendingCommandsClient) Recv() (*PendingCommand, error) {
	m := new(PendingCommand)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *grpcGatewayClient) CancelPendingCommands(ctx context.Context, in *CancelPendingCommandsRequest, opts ...grpc.CallOption) (*CancelPendingCommandsResponse, error) {
	out := new(CancelPendingCommandsResponse)
	err := c.cc.Invoke(ctx, "/ocf.cloud.grpcgateway.pb.GrpcGateway/CancelPendingCommands", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *grpcGatewayClient) CancelPendingMetadataUpdates(ctx context.Context, in *CancelPendingMetadataUpdatesRequest, opts ...grpc.CallOption) (*CancelPendingCommandsResponse, error) {
	out := new(CancelPendingCommandsResponse)
	err := c.cc.Invoke(ctx, "/ocf.cloud.grpcgateway.pb.GrpcGateway/CancelPendingMetadataUpdates", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *grpcGatewayClient) GetDevicesMetadata(ctx context.Context, in *GetDevicesMetadataRequest, opts ...grpc.CallOption) (GrpcGateway_GetDevicesMetadataClient, error) {
	stream, err := c.cc.NewStream(ctx, &GrpcGateway_ServiceDesc.Streams[5], "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetDevicesMetadata", opts...)
	if err != nil {
		return nil, err
	}
	x := &grpcGatewayGetDevicesMetadataClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GrpcGateway_GetDevicesMetadataClient interface {
	Recv() (*events.DeviceMetadataUpdated, error)
	grpc.ClientStream
}

type grpcGatewayGetDevicesMetadataClient struct {
	grpc.ClientStream
}

func (x *grpcGatewayGetDevicesMetadataClient) Recv() (*events.DeviceMetadataUpdated, error) {
	m := new(events.DeviceMetadataUpdated)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *grpcGatewayClient) GetEvents(ctx context.Context, in *GetEventsRequest, opts ...grpc.CallOption) (GrpcGateway_GetEventsClient, error) {
	stream, err := c.cc.NewStream(ctx, &GrpcGateway_ServiceDesc.Streams[6], "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetEvents", opts...)
	if err != nil {
		return nil, err
	}
	x := &grpcGatewayGetEventsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GrpcGateway_GetEventsClient interface {
	Recv() (*GetEventsResponse, error)
	grpc.ClientStream
}

type grpcGatewayGetEventsClient struct {
	grpc.ClientStream
}

func (x *grpcGatewayGetEventsClient) Recv() (*GetEventsResponse, error) {
	m := new(GetEventsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// GrpcGatewayServer is the server API for GrpcGateway service.
// All implementations must embed UnimplementedGrpcGatewayServer
// for forward compatibility
type GrpcGatewayServer interface {
	// Get all devices
	GetDevices(*GetDevicesRequest, GrpcGateway_GetDevicesServer) error
	// Get resource links of devices.
	GetResourceLinks(*GetResourceLinksRequest, GrpcGateway_GetResourceLinksServer) error
	// Get resource from the device.
	GetResourceFromDevice(context.Context, *GetResourceFromDeviceRequest) (*GetResourceFromDeviceResponse, error)
	// Get resources from the resource shadow.
	GetResources(*GetResourcesRequest, GrpcGateway_GetResourcesServer) error
	// Update resource at the device.
	UpdateResource(context.Context, *UpdateResourceRequest) (*UpdateResourceResponse, error)
	// Subscribe to events
	SubscribeToEvents(GrpcGateway_SubscribeToEventsServer) error
	// Get client configuration
	GetClientConfiguration(context.Context, *ClientConfigurationRequest) (*ClientConfigurationResponse, error)
	// Delete resource at the device.
	DeleteResource(context.Context, *DeleteResourceRequest) (*DeleteResourceResponse, error)
	// Create resource at the device.
	CreateResource(context.Context, *CreateResourceRequest) (*CreateResourceResponse, error)
	// Enables/disables shadow synchronization for device.
	UpdateDeviceMetadata(context.Context, *UpdateDeviceMetadataRequest) (*UpdateDeviceMetadataResponse, error)
	// Gets pending commands for devices .
	GetPendingCommands(*GetPendingCommandsRequest, GrpcGateway_GetPendingCommandsServer) error
	// Cancels resource commands.
	CancelPendingCommands(context.Context, *CancelPendingCommandsRequest) (*CancelPendingCommandsResponse, error)
	// Cancels device metadata updates.
	CancelPendingMetadataUpdates(context.Context, *CancelPendingMetadataUpdatesRequest) (*CancelPendingCommandsResponse, error)
	// Gets metadata of the devices. Is contains online/offline or shadown synchronization status.
	GetDevicesMetadata(*GetDevicesMetadataRequest, GrpcGateway_GetDevicesMetadataServer) error
	// Get events for given combination of device id, resource id and timestamp
	GetEvents(*GetEventsRequest, GrpcGateway_GetEventsServer) error
	mustEmbedUnimplementedGrpcGatewayServer()
}

// UnimplementedGrpcGatewayServer must be embedded to have forward compatible implementations.
type UnimplementedGrpcGatewayServer struct {
}

func (UnimplementedGrpcGatewayServer) GetDevices(*GetDevicesRequest, GrpcGateway_GetDevicesServer) error {
	return status.Errorf(codes.Unimplemented, "method GetDevices not implemented")
}
func (UnimplementedGrpcGatewayServer) GetResourceLinks(*GetResourceLinksRequest, GrpcGateway_GetResourceLinksServer) error {
	return status.Errorf(codes.Unimplemented, "method GetResourceLinks not implemented")
}
func (UnimplementedGrpcGatewayServer) GetResourceFromDevice(context.Context, *GetResourceFromDeviceRequest) (*GetResourceFromDeviceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetResourceFromDevice not implemented")
}
func (UnimplementedGrpcGatewayServer) GetResources(*GetResourcesRequest, GrpcGateway_GetResourcesServer) error {
	return status.Errorf(codes.Unimplemented, "method GetResources not implemented")
}
func (UnimplementedGrpcGatewayServer) UpdateResource(context.Context, *UpdateResourceRequest) (*UpdateResourceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateResource not implemented")
}
func (UnimplementedGrpcGatewayServer) SubscribeToEvents(GrpcGateway_SubscribeToEventsServer) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeToEvents not implemented")
}
func (UnimplementedGrpcGatewayServer) GetClientConfiguration(context.Context, *ClientConfigurationRequest) (*ClientConfigurationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetClientConfiguration not implemented")
}
func (UnimplementedGrpcGatewayServer) DeleteResource(context.Context, *DeleteResourceRequest) (*DeleteResourceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteResource not implemented")
}
func (UnimplementedGrpcGatewayServer) CreateResource(context.Context, *CreateResourceRequest) (*CreateResourceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateResource not implemented")
}
func (UnimplementedGrpcGatewayServer) UpdateDeviceMetadata(context.Context, *UpdateDeviceMetadataRequest) (*UpdateDeviceMetadataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateDeviceMetadata not implemented")
}
func (UnimplementedGrpcGatewayServer) GetPendingCommands(*GetPendingCommandsRequest, GrpcGateway_GetPendingCommandsServer) error {
	return status.Errorf(codes.Unimplemented, "method GetPendingCommands not implemented")
}
func (UnimplementedGrpcGatewayServer) CancelPendingCommands(context.Context, *CancelPendingCommandsRequest) (*CancelPendingCommandsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CancelPendingCommands not implemented")
}
func (UnimplementedGrpcGatewayServer) CancelPendingMetadataUpdates(context.Context, *CancelPendingMetadataUpdatesRequest) (*CancelPendingCommandsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CancelPendingMetadataUpdates not implemented")
}
func (UnimplementedGrpcGatewayServer) GetDevicesMetadata(*GetDevicesMetadataRequest, GrpcGateway_GetDevicesMetadataServer) error {
	return status.Errorf(codes.Unimplemented, "method GetDevicesMetadata not implemented")
}
func (UnimplementedGrpcGatewayServer) GetEvents(*GetEventsRequest, GrpcGateway_GetEventsServer) error {
	return status.Errorf(codes.Unimplemented, "method GetEvents not implemented")
}
func (UnimplementedGrpcGatewayServer) mustEmbedUnimplementedGrpcGatewayServer() {}

// UnsafeGrpcGatewayServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GrpcGatewayServer will
// result in compilation errors.
type UnsafeGrpcGatewayServer interface {
	mustEmbedUnimplementedGrpcGatewayServer()
}

func RegisterGrpcGatewayServer(s grpc.ServiceRegistrar, srv GrpcGatewayServer) {
	s.RegisterService(&GrpcGateway_ServiceDesc, srv)
}

func _GrpcGateway_GetDevices_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetDevicesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GrpcGatewayServer).GetDevices(m, &grpcGatewayGetDevicesServer{stream})
}

type GrpcGateway_GetDevicesServer interface {
	Send(*Device) error
	grpc.ServerStream
}

type grpcGatewayGetDevicesServer struct {
	grpc.ServerStream
}

func (x *grpcGatewayGetDevicesServer) Send(m *Device) error {
	return x.ServerStream.SendMsg(m)
}

func _GrpcGateway_GetResourceLinks_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetResourceLinksRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GrpcGatewayServer).GetResourceLinks(m, &grpcGatewayGetResourceLinksServer{stream})
}

type GrpcGateway_GetResourceLinksServer interface {
	Send(*events.ResourceLinksPublished) error
	grpc.ServerStream
}

type grpcGatewayGetResourceLinksServer struct {
	grpc.ServerStream
}

func (x *grpcGatewayGetResourceLinksServer) Send(m *events.ResourceLinksPublished) error {
	return x.ServerStream.SendMsg(m)
}

func _GrpcGateway_GetResourceFromDevice_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetResourceFromDeviceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GrpcGatewayServer).GetResourceFromDevice(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetResourceFromDevice",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GrpcGatewayServer).GetResourceFromDevice(ctx, req.(*GetResourceFromDeviceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GrpcGateway_GetResources_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetResourcesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GrpcGatewayServer).GetResources(m, &grpcGatewayGetResourcesServer{stream})
}

type GrpcGateway_GetResourcesServer interface {
	Send(*Resource) error
	grpc.ServerStream
}

type grpcGatewayGetResourcesServer struct {
	grpc.ServerStream
}

func (x *grpcGatewayGetResourcesServer) Send(m *Resource) error {
	return x.ServerStream.SendMsg(m)
}

func _GrpcGateway_UpdateResource_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateResourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GrpcGatewayServer).UpdateResource(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ocf.cloud.grpcgateway.pb.GrpcGateway/UpdateResource",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GrpcGatewayServer).UpdateResource(ctx, req.(*UpdateResourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GrpcGateway_SubscribeToEvents_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(GrpcGatewayServer).SubscribeToEvents(&grpcGatewaySubscribeToEventsServer{stream})
}

type GrpcGateway_SubscribeToEventsServer interface {
	Send(*Event) error
	Recv() (*SubscribeToEvents, error)
	grpc.ServerStream
}

type grpcGatewaySubscribeToEventsServer struct {
	grpc.ServerStream
}

func (x *grpcGatewaySubscribeToEventsServer) Send(m *Event) error {
	return x.ServerStream.SendMsg(m)
}

func (x *grpcGatewaySubscribeToEventsServer) Recv() (*SubscribeToEvents, error) {
	m := new(SubscribeToEvents)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _GrpcGateway_GetClientConfiguration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClientConfigurationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GrpcGatewayServer).GetClientConfiguration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetClientConfiguration",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GrpcGatewayServer).GetClientConfiguration(ctx, req.(*ClientConfigurationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GrpcGateway_DeleteResource_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteResourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GrpcGatewayServer).DeleteResource(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ocf.cloud.grpcgateway.pb.GrpcGateway/DeleteResource",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GrpcGatewayServer).DeleteResource(ctx, req.(*DeleteResourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GrpcGateway_CreateResource_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateResourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GrpcGatewayServer).CreateResource(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ocf.cloud.grpcgateway.pb.GrpcGateway/CreateResource",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GrpcGatewayServer).CreateResource(ctx, req.(*CreateResourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GrpcGateway_UpdateDeviceMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateDeviceMetadataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GrpcGatewayServer).UpdateDeviceMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ocf.cloud.grpcgateway.pb.GrpcGateway/UpdateDeviceMetadata",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GrpcGatewayServer).UpdateDeviceMetadata(ctx, req.(*UpdateDeviceMetadataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GrpcGateway_GetPendingCommands_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetPendingCommandsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GrpcGatewayServer).GetPendingCommands(m, &grpcGatewayGetPendingCommandsServer{stream})
}

type GrpcGateway_GetPendingCommandsServer interface {
	Send(*PendingCommand) error
	grpc.ServerStream
}

type grpcGatewayGetPendingCommandsServer struct {
	grpc.ServerStream
}

func (x *grpcGatewayGetPendingCommandsServer) Send(m *PendingCommand) error {
	return x.ServerStream.SendMsg(m)
}

func _GrpcGateway_CancelPendingCommands_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CancelPendingCommandsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GrpcGatewayServer).CancelPendingCommands(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ocf.cloud.grpcgateway.pb.GrpcGateway/CancelPendingCommands",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GrpcGatewayServer).CancelPendingCommands(ctx, req.(*CancelPendingCommandsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GrpcGateway_CancelPendingMetadataUpdates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CancelPendingMetadataUpdatesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GrpcGatewayServer).CancelPendingMetadataUpdates(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ocf.cloud.grpcgateway.pb.GrpcGateway/CancelPendingMetadataUpdates",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GrpcGatewayServer).CancelPendingMetadataUpdates(ctx, req.(*CancelPendingMetadataUpdatesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GrpcGateway_GetDevicesMetadata_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetDevicesMetadataRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GrpcGatewayServer).GetDevicesMetadata(m, &grpcGatewayGetDevicesMetadataServer{stream})
}

type GrpcGateway_GetDevicesMetadataServer interface {
	Send(*events.DeviceMetadataUpdated) error
	grpc.ServerStream
}

type grpcGatewayGetDevicesMetadataServer struct {
	grpc.ServerStream
}

func (x *grpcGatewayGetDevicesMetadataServer) Send(m *events.DeviceMetadataUpdated) error {
	return x.ServerStream.SendMsg(m)
}

func _GrpcGateway_GetEvents_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetEventsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GrpcGatewayServer).GetEvents(m, &grpcGatewayGetEventsServer{stream})
}

type GrpcGateway_GetEventsServer interface {
	Send(*GetEventsResponse) error
	grpc.ServerStream
}

type grpcGatewayGetEventsServer struct {
	grpc.ServerStream
}

func (x *grpcGatewayGetEventsServer) Send(m *GetEventsResponse) error {
	return x.ServerStream.SendMsg(m)
}

// GrpcGateway_ServiceDesc is the grpc.ServiceDesc for GrpcGateway service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var GrpcGateway_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ocf.cloud.grpcgateway.pb.GrpcGateway",
	HandlerType: (*GrpcGatewayServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetResourceFromDevice",
			Handler:    _GrpcGateway_GetResourceFromDevice_Handler,
		},
		{
			MethodName: "UpdateResource",
			Handler:    _GrpcGateway_UpdateResource_Handler,
		},
		{
			MethodName: "GetClientConfiguration",
			Handler:    _GrpcGateway_GetClientConfiguration_Handler,
		},
		{
			MethodName: "DeleteResource",
			Handler:    _GrpcGateway_DeleteResource_Handler,
		},
		{
			MethodName: "CreateResource",
			Handler:    _GrpcGateway_CreateResource_Handler,
		},
		{
			MethodName: "UpdateDeviceMetadata",
			Handler:    _GrpcGateway_UpdateDeviceMetadata_Handler,
		},
		{
			MethodName: "CancelPendingCommands",
			Handler:    _GrpcGateway_CancelPendingCommands_Handler,
		},
		{
			MethodName: "CancelPendingMetadataUpdates",
			Handler:    _GrpcGateway_CancelPendingMetadataUpdates_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetDevices",
			Handler:       _GrpcGateway_GetDevices_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetResourceLinks",
			Handler:       _GrpcGateway_GetResourceLinks_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetResources",
			Handler:       _GrpcGateway_GetResources_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SubscribeToEvents",
			Handler:       _GrpcGateway_SubscribeToEvents_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "GetPendingCommands",
			Handler:       _GrpcGateway_GetPendingCommands_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetDevicesMetadata",
			Handler:       _GrpcGateway_GetDevicesMetadata_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetEvents",
			Handler:       _GrpcGateway_GetEvents_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "github.com/plgd-dev/cloud/grpc-gateway/pb/service.proto",
}
