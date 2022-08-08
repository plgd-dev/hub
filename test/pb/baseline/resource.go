package baseline

import (
	"testing"

	"github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
)

func CmpResourceChangedData(t *testing.T, expected, got []byte) {
	cleanUpLinks := func(data map[interface{}]interface{}) {
		if data == nil {
			return
		}
		links, ok := data["links"].([]interface{})
		if !ok {
			return
		}
		for i := range links {
			link, ok := links[i].(map[interface{}]interface{})
			if !ok {
				continue
			}
			delete(link, "eps")
			delete(link, "ins")
			delete(data, "pi")
			delete(data, "piid")
		}
	}

	gotData, ok := test.DecodeCbor(t, got).(map[interface{}]interface{})
	require.True(t, ok)
	expectedData, ok := test.DecodeCbor(t, expected).(map[interface{}]interface{})
	require.True(t, ok)
	cleanUpLinks(expectedData)
	cleanUpLinks(gotData)
	require.Equal(t, expectedData, gotData)
}
