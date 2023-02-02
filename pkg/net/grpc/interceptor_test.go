package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnaryInterceptor(t *testing.T) {
	m := &MockInterceptor{}
	svr := StubGrpcServer(grpc.UnaryServerInterceptorOption(m.Intercept))
	defer func() {
		_ = svr.Close()
	}()
	go func() {
		_ = svr.Serve()
	}()
	time.Sleep(100 * time.Millisecond)
	c := StubGrpcClient(svr.Addr())
	_, err := c.TestCall(context.Background(), &TestRequest{})
	require.Error(t, err)
	assert.Equal(t, "/"+StubService_ServiceDesc.ServiceName+"/TestCall", m.Method)
}

func TestStreamInterceptor(t *testing.T) {
	m := &MockInterceptor{}
	svr := StubGrpcServer(grpc.StreamServerInterceptorOption(m.Intercept))
	defer func() {
		_ = svr.Close()
	}()
	go func() {
		_ = svr.Serve()
	}()
	time.Sleep(100 * time.Millisecond)
	c := StubGrpcClient(svr.Addr())
	s, err := c.TestStream(context.Background())
	require.NoError(t, err)
	err = s.Send(&TestRequest{})
	require.NoError(t, err)
	_, err = s.Recv()
	require.Error(t, err)
	assert.Equal(t, "/"+StubService_ServiceDesc.ServiceName+"/TestStream", m.Method)
}

type MockInterceptor struct {
	Method string
}

func (i *MockInterceptor) Intercept(ctx context.Context, method string) (context.Context, error) {
	i.Method = method
	return ctx, nil
}
