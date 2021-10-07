package grpc_test

import (
	"github.com/plgd-dev/hub/pkg/net/grpc/server"
	"google.golang.org/grpc"
)

func StubGrpcServer(opts ...grpc.ServerOption) *server.Server {
	svr, err := server.NewServer(":", opts...)
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
