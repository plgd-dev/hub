package test

import (
	"testing"

	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/stretchr/testify/require"
)

func CmpResourceCreated(t *testing.T, expected, got *events.ResourceCreated) {
	require.NotEmpty(t, got)

	gotData, ok := DecodeCbor(t, got.GetContent().GetData()).(map[interface{}]interface{})
	require.True(t, ok)
	delete(gotData, "ins") // instance_id is a random value
	expectedData, ok := DecodeCbor(t, expected.GetContent().GetData()).(map[interface{}]interface{})
	require.True(t, ok)
	delete(gotData, "ins")
	require.Equal(t, gotData, expectedData)
	got.GetContent().Data = nil
	expected.GetContent().Data = nil

	expected.AuditContext = nil
	got.AuditContext = nil
	expected.EventMetadata = nil
	got.EventMetadata = nil

	CheckProtobufs(t, expected, got, RequireToCheckFunc(require.Equal))
}
