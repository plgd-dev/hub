// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.3
// source: github.com/plgd-dev/hub/pkg/net/grpc/stub.proto

package grpc_test

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
	StubService_TestCall_FullMethodName   = "/pkg.net.grpc.server.StubService/TestCall"
	StubService_TestStream_FullMethodName = "/pkg.net.grpc.server.StubService/TestStream"
)

// StubServiceClient is the client API for StubService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StubServiceClient interface {
	TestCall(ctx context.Context, in *TestRequest, opts ...grpc.CallOption) (*TestResponse, error)
	TestStream(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[TestRequest, TestResponse], error)
}

type stubServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewStubServiceClient(cc grpc.ClientConnInterface) StubServiceClient {
	return &stubServiceClient{cc}
}

func (c *stubServiceClient) TestCall(ctx context.Context, in *TestRequest, opts ...grpc.CallOption) (*TestResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TestResponse)
	err := c.cc.Invoke(ctx, StubService_TestCall_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *stubServiceClient) TestStream(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[TestRequest, TestResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &StubService_ServiceDesc.Streams[0], StubService_TestStream_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[TestRequest, TestResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type StubService_TestStreamClient = grpc.BidiStreamingClient[TestRequest, TestResponse]

// StubServiceServer is the server API for StubService service.
// All implementations must embed UnimplementedStubServiceServer
// for forward compatibility.
type StubServiceServer interface {
	TestCall(context.Context, *TestRequest) (*TestResponse, error)
	TestStream(grpc.BidiStreamingServer[TestRequest, TestResponse]) error
	mustEmbedUnimplementedStubServiceServer()
}

// UnimplementedStubServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedStubServiceServer struct{}

func (UnimplementedStubServiceServer) TestCall(context.Context, *TestRequest) (*TestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TestCall not implemented")
}
func (UnimplementedStubServiceServer) TestStream(grpc.BidiStreamingServer[TestRequest, TestResponse]) error {
	return status.Errorf(codes.Unimplemented, "method TestStream not implemented")
}
func (UnimplementedStubServiceServer) mustEmbedUnimplementedStubServiceServer() {}
func (UnimplementedStubServiceServer) testEmbeddedByValue()                     {}

// UnsafeStubServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StubServiceServer will
// result in compilation errors.
type UnsafeStubServiceServer interface {
	mustEmbedUnimplementedStubServiceServer()
}

func RegisterStubServiceServer(s grpc.ServiceRegistrar, srv StubServiceServer) {
	// If the following call pancis, it indicates UnimplementedStubServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&StubService_ServiceDesc, srv)
}

func _StubService_TestCall_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StubServiceServer).TestCall(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StubService_TestCall_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StubServiceServer).TestCall(ctx, req.(*TestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _StubService_TestStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(StubServiceServer).TestStream(&grpc.GenericServerStream[TestRequest, TestResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type StubService_TestStreamServer = grpc.BidiStreamingServer[TestRequest, TestResponse]

// StubService_ServiceDesc is the grpc.ServiceDesc for StubService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var StubService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pkg.net.grpc.server.StubService",
	HandlerType: (*StubServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "TestCall",
			Handler:    _StubService_TestCall_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "TestStream",
			Handler:       _StubService_TestStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "github.com/plgd-dev/hub/pkg/net/grpc/stub.proto",
}
