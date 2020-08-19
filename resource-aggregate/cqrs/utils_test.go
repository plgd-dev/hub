package cqrs

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/stretchr/testify/assert"
)

func TestDummyForCoverage(t *testing.T) {
	deviceID := "dev"

	GetTopics(deviceID)
	MakeResourceId(deviceID, "/abc")
	sequence := uint64(1234)
	version := uint64(5)
	connId := "c"
	corID := "a"
	userID := "u"

	TimeNowMs()
	em := MakeEventMeta(connId, sequence, version)
	assert.Equal(t, connId, em.ConnectionId)
	assert.Equal(t, sequence, em.Sequence)
	assert.Equal(t, version, em.Version)
	ac := MakeAuditContext(deviceID, userID, corID)
	assert.Equal(t, corID, ac.CorrelationId)
	assert.Equal(t, userID, ac.UserId)
	assert.Equal(t, deviceID, ac.DeviceId)
}

func TestProtobufMarshaler(t *testing.T) {
	req := pb.AuthorizationContext{}

	out, err := Marshal(&req)
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	a := struct {
	}{}
	_, err = Marshal(a)
	assert.Error(t, err)

	resp := pb.AuthorizationContext{}
	err = Unmarshal(out, &resp)
	assert.NoError(t, err)

	err = Unmarshal(out, a)
	assert.Error(t, err)
}
