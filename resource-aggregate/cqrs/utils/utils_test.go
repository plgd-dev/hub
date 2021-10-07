package utils_test

import (
	"testing"

	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/stretchr/testify/assert"
)

func TestDummyForCoverage(t *testing.T) {
	deviceID := "dev"

	utils.GetDeviceSubject("a", deviceID)
	sequence := uint64(1234)
	version := uint64(5)
	connId := "c"
	corID := "a"
	userID := "u"

	utils.TimeNowMs()
	em := events.MakeEventMeta(connId, sequence, version)
	assert.Equal(t, connId, em.ConnectionId)
	assert.Equal(t, sequence, em.Sequence)
	assert.Equal(t, version, em.Version)
	ac := commands.NewAuditContext(userID, corID)
	assert.Equal(t, corID, ac.CorrelationId)
	assert.Equal(t, userID, ac.UserId)
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
