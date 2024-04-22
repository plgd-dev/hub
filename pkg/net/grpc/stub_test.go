package grpc_test

import (
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	return NewStubServiceClient(conn)
}
