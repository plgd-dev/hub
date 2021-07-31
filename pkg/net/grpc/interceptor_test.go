package grpc_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnaryInterceptor(t *testing.T) {
	m := &MockInterceptor{}
	svr := server.StubGrpcServer(grpc.UnaryServerInterceptorOption(m.Intercept))
	defer svr.Close()
	go func() {
		_ = svr.Serve()
	}()

	c := server.StubGrpcClient(svr.Addr())
	_, err := c.TestCall(context.Background(), &server.TestRequest{})
	require.Error(t, err)
	assert.Equal(t, "/ocf.cloud.test.pb.StubService/TestCall", m.Method)
}

func TestStreamInterceptor(t *testing.T) {
	m := &MockInterceptor{}
	svr := server.StubGrpcServer(grpc.StreamServerInterceptorOption(m.Intercept))
	defer svr.Close()
	go func() {
		_ = svr.Serve()
	}()

	c := server.StubGrpcClient(svr.Addr())
	s, err := c.TestStream(context.Background())
	require.NoError(t, err)
	err = s.Send(&server.TestRequest{})
	require.NoError(t, err)
	_, err = s.Recv()
	require.Error(t, err)
	assert.Equal(t, "/ocf.cloud.test.pb.StubService/TestStream", m.Method)
}

type MockInterceptor struct {
	Method string
}

func (i *MockInterceptor) Intercept(ctx context.Context, method string) (context.Context, error) {
	i.Method = method
	return ctx, nil
}
