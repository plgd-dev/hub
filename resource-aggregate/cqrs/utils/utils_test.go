package utils_test

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/stretchr/testify/assert"
)

func TestDummyForCoverage(t *testing.T) {
	deviceID := "dev"

	utils.GetTopics(deviceID)
	utils.MakeResourceId(deviceID, "/abc")
	sequence := uint64(1234)
	version := uint64(5)
	connId := "c"
	corID := "a"
	userID := "u"

	utils.TimeNowMs()
	em := utils.MakeEventMeta(connId, sequence, version)
	assert.Equal(t, connId, em.ConnectionId)
	assert.Equal(t, sequence, em.Sequence)
	assert.Equal(t, version, em.Version)
	ac := utils.MakeAuditContext(deviceID, userID, corID)
	assert.Equal(t, corID, ac.CorrelationId)
	assert.Equal(t, userID, ac.UserId)
	assert.Equal(t, deviceID, ac.DeviceId)
}

func TestProtobufMarshaler(t *testing.T) {
	req := events.ResourceChanged{}

	out, err := utils.Marshal(&req)
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	a := struct {
	}{}
	_, err = utils.Marshal(a)
	assert.Error(t, err)

	resp := events.ResourceChanged{}
	err = utils.Unmarshal(out, &resp)
	assert.NoError(t, err)

	err = utils.Unmarshal(out, a)
	assert.Error(t, err)
}
