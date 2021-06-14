package service_test

import (
	"testing"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-directory/service"
	"github.com/plgd-dev/cloud/test"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "simple map",
			args: args{
				v: map[string]interface{}{
					"a": 1,
					"b": "c",
					"d": 1.01,
				},
			},
			want: []byte(`{"a":1,"b":"c","d":1.01}`),
		},
		{
			name: "simple struct",
			args: args{
				v: struct {
					A int    `json:"a"`
					B string `json:"BB"`
					D float64
				}{
					A: 1,
					B: "c",
					D: 1.01,
				},
			},
			want: []byte(`{"a":1,"BB":"c","D":1.01}`),
		},
		{
			name: "protobuf simple",
			args: args{
				v: commands.ResourceId{
					DeviceId: "1",
					Href:     "/oic/d",
				},
			},
			want: []byte(`{"deviceId":"1","href":"/oic/d"}`),
		},
		{
			name: "protobuf array",
			args: args{
				v: []*pb.Device{
					{
						Types:      []string{"oic.d.cloudDevice", "oic.wk.d"},
						Interfaces: []string{"oic.if.r", "oic.if.baseline"},
						Id:         "1",
						Name:       test.TestDeviceName,
						Metadata: &pb.Device_Metadata{
							Status: &commands.ConnectionStatus{
								Value: commands.ConnectionStatus_ONLINE,
							},
						},
					},
				},
			},
			want: []byte(`[{"id":"1","types":["oic.d.cloudDevice","oic.wk.d"],"name":"` + test.TestDeviceName + `","metadata":{"status":{"value":1}},"interfaces":["oic.if.r","oic.if.baseline"]}]`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.Encode(tt.args.v)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, string(tt.want), string(got))
		})
	}
}

func TestDecode(t *testing.T) {
	type args struct {
		data []byte
		v    interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		{
			name: "protobuf",
			args: args{
				data: []byte(`{"deviceId":"1","href":"/oic/d"}`),
				v:    new(commands.ResourceId),
			},
			want: &commands.ResourceId{
				DeviceId: "1",
				Href:     "/oic/d",
			},
		},
		{
			name: "protobuf array",
			args: args{
				data: []byte(`[{"id":"1","types":["oic.d.cloudDevice","oic.wk.d"],"name":"` + test.TestDeviceName + `","metadata":{"status":{"value":1}},"interfaces":["oic.if.r","oic.if.baseline"]}]`),
				v:    new([]*pb.Device),
			},
			want: []*pb.Device{
				{
					Types:      []string{"oic.d.cloudDevice", "oic.wk.d"},
					Interfaces: []string{"oic.if.r", "oic.if.baseline"},
					Id:         "1",
					Name:       test.TestDeviceName,
					Metadata: &pb.Device_Metadata{
						Status: &commands.ConnectionStatus{
							Value: commands.ConnectionStatus_ONLINE,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Decode(tt.args.data, tt.args.v)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			test.CheckProtobufs(t, tt.want, tt.args.v, test.RequireToCheckFunc(require.Equal))
		})
	}
}
