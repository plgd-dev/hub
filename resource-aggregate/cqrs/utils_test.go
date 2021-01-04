package cqrs_test

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/stretchr/testify/assert"
)

func TestDummyForCoverage(t *testing.T) {
	deviceID := "dev"

	cqrs.GetTopics(deviceID)
	cqrs.MakeResourceId(deviceID, "/abc")
	sequence := uint64(1234)
	version := uint64(5)
	connId := "c"
	corID := "a"
	userID := "u"

	cqrs.TimeNowMs()
	em := cqrs.MakeEventMeta(connId, sequence, version)
	assert.Equal(t, connId, em.ConnectionId)
	assert.Equal(t, sequence, em.Sequence)
	assert.Equal(t, version, em.Version)
	ac := cqrs.MakeAuditContext(deviceID, userID, corID)
	assert.Equal(t, corID, ac.CorrelationId)
	assert.Equal(t, userID, ac.UserId)
	assert.Equal(t, deviceID, ac.DeviceId)
}

func TestProtobufMarshaler(t *testing.T) {
	req := events.ResourceChanged{}

	out, err := cqrs.Marshal(&req)
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	a := struct {
	}{}
	_, err = cqrs.Marshal(a)
	assert.Error(t, err)

	resp := events.ResourceChanged{}
	err = cqrs.Unmarshal(out, &resp)
	assert.NoError(t, err)

	err = cqrs.Unmarshal(out, a)
	assert.Error(t, err)
}
