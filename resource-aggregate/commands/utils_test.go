package commands_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestResourceIdFromString(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name string
		args args
		want *commands.ResourceId
	}{
		{
			name: "Nil",
			args: args{
				v: "",
			},
			want: nil,
		},
		{
			name: "Simple",
			args: args{
				v: "deviceId/href",
			},
			want: &commands.ResourceId{
				DeviceId: "deviceId",
				Href:     "/href",
			},
		},
		{
			name: "deviceId",
			args: args{
				v: "deviceId/",
			},
			want: &commands.ResourceId{
				DeviceId: "deviceId",
				Href:     "/",
			},
		},
		{
			name: "hrefId",
			args: args{
				v: "//hrefId",
			},
			want: &commands.ResourceId{
				DeviceId: "",
				Href:     "/hrefId",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commands.ResourceIdFromString(tt.args.v)
			require.True(t, proto.Equal(tt.want, got))
		})
	}
}

func TestDecodeContent(t *testing.T) {
	type args struct {
		content *commands.Content
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "nil",
			args: args{
				content: nil,
			},
			wantErr: true,
		},
		{
			name: "unsupported type",
			args: args{
				content: &commands.Content{
					ContentType: "unsupported",
					Data:        []byte("test"),
				},
			},
			wantErr: true,
		},
		{
			name: "cbor",
			args: args{
				content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data:        test.EncodeToCbor(t, map[string]interface{}{"test": "test"}),
				},
			},
			want: map[string]interface{}{"test": "test"},
		},
		{
			name: "json",
			args: args{
				content: &commands.Content{
					ContentType: message.AppJSON.String(),
					Data: func() []byte {
						d, err := json.Encode(map[string]interface{}{"test": "test"})
						require.NoError(t, err)
						return d
					}(),
				},
			},
			want: map[string]interface{}{"test": "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]interface{}
			err := commands.DecodeContent(tt.args.content, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeTextContent(t *testing.T) {
	content := &commands.Content{
		ContentType: message.TextPlain.String(),
		Data:        []byte("test"),
	}
	var gotStr string
	err := commands.DecodeContent(content, &gotStr)
	require.NoError(t, err)
	require.Equal(t, "test", gotStr)

	var gotBytes []byte
	err = commands.DecodeContent(content, &gotBytes)
	require.NoError(t, err)
	require.Equal(t, []byte("test"), gotBytes)

	var got interface{}
	err = commands.DecodeContent(content, &got)
	require.NoError(t, err)
	str, ok := got.(string)
	require.True(t, ok)
	require.Equal(t, "test", str)

	var gotMap map[string]interface{}
	err = commands.DecodeContent(content, &gotMap)
	require.Error(t, err)
}
