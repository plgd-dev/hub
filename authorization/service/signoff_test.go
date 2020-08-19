package service

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignOff(t *testing.T) {
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, newTestDevice())

	_, err := s.service.SignOff(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignOffRequest())
	assert := assert.New(t)
	assert.NoError(err)
	_, ok := retrieveDevice(t, s.service.persistence, testDeviceID, testUserID)
	assert.False(ok)
}

func TestNonexistentDevice(t *testing.T) {
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()

	_, err := s.service.SignOff(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignOffRequest())
	require.Error(t, err)
}

func TestUnexpectedAccessTokenOnSignOff(t *testing.T) {
	d := newTestDevice()
	d.AccessToken = "unexpected"
	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, d)

	_, err := s.service.SignOff(kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken), newSignOffRequest())
	assert := assert.New(t)
	assert.Error(err)
}

func newSignOffRequest() *pb.SignOffRequest {
	return &pb.SignOffRequest{
		DeviceId:    testDeviceID,
		UserId:      testUserID,
		AccessToken: testAccessToken,
	}
}
