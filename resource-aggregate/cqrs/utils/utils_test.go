package utils_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDummyForCoverage(t *testing.T) {
	deviceID := "dev"

	utils.GetDeviceSubject("a", deviceID)
	sequence := uint64(1234)
	version := uint64(5)
	connID := "c"
	corID := "a"
	userID := "u"
	hubID := "h"

	em := events.MakeEventMeta(connID, sequence, version, hubID)
	assert.Equal(t, connID, em.ConnectionId)
	assert.Equal(t, sequence, em.Sequence)
	assert.Equal(t, version, em.Version)
	assert.Equal(t, hubID, em.HubId)
	ac := commands.NewAuditContext(userID, corID, userID)
	assert.Equal(t, corID, ac.CorrelationId)
	assert.Equal(t, userID, ac.UserId)
}

func TestProtobufMarshaler(t *testing.T) {
	req := events.ResourceChanged{}

	out, err := utils.Marshal(&req)
	require.NoError(t, err)
	assert.NotEmpty(t, out)

	a := struct{}{}
	_, err = utils.Marshal(a)
	require.Error(t, err)

	resp := events.ResourceChanged{}
	err = utils.Unmarshal(out, &resp)
	require.NoError(t, err)

	err = utils.Unmarshal(out, a)
	require.Error(t, err)
}
