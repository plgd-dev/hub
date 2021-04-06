package server

import (
	"google.golang.org/grpc"
)

func StubGrpcServer(opts ...grpc.ServerOption) *Server {
	svr, err := NewServer(":", opts...)
	if err != nil {
		panic(err)
	}
	handler := UnimplementedStubServiceServer{}
	RegisterStubServiceServer(svr.Server, &handler)
	return svr
}

func StubGrpcClient(addr string) StubServiceClient {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	return NewStubServiceClient(conn)
}
