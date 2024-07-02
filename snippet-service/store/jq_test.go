package store_test

import (
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

func TestEvalJQCondition(t *testing.T) {
	tests := []struct {
		name    string
		jq      string
		content *commands.Content
		wantErr bool
		want    bool
	}{
		{
			name: "invalid jq",
			jq:   "][",
			content: &commands.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: hubTest.EncodeToCbor(t, map[string]interface{}{
					"value": 42,
				}),
			},
			wantErr: true,
		},
		{
			name: "invalid jq returned type", // we expect a boolean
			jq:   ".",
			content: &commands.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: hubTest.EncodeToCbor(t, map[string]interface{}{
					"value": 42,
				}),
			},
			wantErr: true,
		},
		{
			name: "nonexisting attribute",
			jq:   ".nonexisting == 42",
			content: &commands.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: hubTest.EncodeToCbor(t, map[string]interface{}{
					"value": 42,
				}),
			},
			want: false,
		},
		{
			name: "int",
			jq:   ". == 42",
			content: &commands.Content{
				ContentType: message.AppOcfCbor.String(),
				Data:        hubTest.EncodeToCbor(t, 42),
			},
			want: true,
		},
		{
			name: "string",
			jq:   ". == \"leet\"",
			content: &commands.Content{
				ContentType: message.AppJSON.String(),
				Data: func() []byte {
					data, err := json.Encode("leet")
					require.NoError(t, err)
					return data
				}(),
			},
			want: true,
		},
		{
			name: "array",
			jq:   ". == [1, 2, 3]",
			content: &commands.Content{
				ContentType: message.AppJSON.String(),
				Data: func() []byte {
					data, err := json.Encode([]int{1, 2, 3})
					require.NoError(t, err)
					return data
				}(),
			},
			want: true,
		},
		{
			name: "object",
			jq:   ".a == 42 and .b == \"leet\" and .c[1] == 2",
			content: &commands.Content{
				ContentType: message.AppJSON.String(),
				Data: func() []byte {
					data, err := json.Encode(map[string]any{
						"a": 42,
						"b": "leet",
						"c": []int{1, 2, 3},
					})
					require.NoError(t, err)
					return data
				}(),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var json any
			var jsonMap map[string]any
			err := commands.DecodeContent(tt.content, &jsonMap)
			if err == nil {
				json = jsonMap
			} else {
				err = commands.DecodeContent(tt.content, &json)
			}
			require.NoError(t, err)
			got, err := store.EvalJQCondition(tt.jq, json)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
