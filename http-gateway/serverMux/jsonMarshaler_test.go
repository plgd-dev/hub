package serverMux_test

import (
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

func TestJsonMarshalerMarshal(t *testing.T) {
	type fields struct {
		JSONPb *runtime.JSONPb
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				v: pb.MakeResourceChanged(t, "deviceID", "/href", "correlationID", map[interface{}]interface{}{
					"key": "value",
				}),
			},
			want: map[interface{}]interface{}{
				"auditContext": map[interface{}]interface{}{
					"correlationId": "correlationID",
					"userId":        "1",
				},
				"content": map[interface{}]interface{}{
					"key": "value",
				},
				"resourceId": map[interface{}]interface{}{
					"deviceId": "deviceID",
					"href":     "/href",
				},
				"status": "OK",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := serverMux.NewJsonMarshaler()
			got, err := j.Marshal(tt.args.v)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			var v interface{}
			err = json.Decode(got, &v)
			require.NoError(t, err)
			require.Equal(t, tt.want, v)
		})
	}
}
