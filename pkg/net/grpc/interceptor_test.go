package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnaryInterceptor(t *testing.T) {
	m := &MockInterceptor{}
	svr := StubGrpcServer(UnaryServerInterceptorOption(m.Intercept))
	defer svr.Close()
	go svr.Serve()

	c := StubGrpcClient(svr.Addr())
	c.TestCall(context.Background(), &TestRequest{})
	assert.Equal(t, "/ocf.cloud.test.pb.StubService/TestCall", m.Method)
}

func TestStreamInterceptor(t *testing.T) {
	m := &MockInterceptor{}
	svr := StubGrpcServer(StreamServerInterceptorOption(m.Intercept))
	defer svr.Close()
	go svr.Serve()

	c := StubGrpcClient(svr.Addr())
	s, err := c.TestStream(context.Background())
	require.NoError(t, err)
	err = s.Send(&TestRequest{})
	require.NoError(t, err)
	s.Recv()
	assert.Equal(t, "/ocf.cloud.test.pb.StubService/TestStream", m.Method)
}

type MockInterceptor struct {
	Method string
}

func (i *MockInterceptor) Intercept(ctx context.Context, method string) (context.Context, error) {
	i.Method = method
	return ctx, nil
}
