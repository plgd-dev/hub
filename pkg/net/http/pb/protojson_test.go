package pb_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/status"
	grpcStatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestUnmarshalError(t *testing.T) {
	s := &status.Status{
		Code:    http.StatusInternalServerError,
		Message: "test error",
	}
	data, err := protojson.Marshal(s)
	require.NoError(t, err)

	err = pb.UnmarshalError(data)
	require.Error(t, err)

	st, ok := grpcStatus.FromError(err)
	require.True(t, ok)
	require.Equal(t, s.GetCode(), int32(st.Code()))
	require.Equal(t, s.GetMessage(), st.Message())
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name          string
		code          int
		input         []byte
		wantGrpcError error
		wantErr       bool
		want          protoreflect.ProtoMessage
	}{
		{
			name: "Unmarshal success",
			code: http.StatusOK,
			input: func() []byte {
				data, err := protojson.Marshal(structpb.NewStringValue("test"))
				require.NoError(t, err)
				return []byte(`{"result":` + string(data) + `}`)
			}(),
			want: structpb.NewStringValue("test"),
		},
		{
			name:  "Unmarshal error status",
			code:  http.StatusInternalServerError,
			input: []byte(`{"code": 500, "message": "test error"}`),
			wantGrpcError: grpcStatus.ErrorProto(&status.Status{
				Code:    http.StatusInternalServerError,
				Message: "test error",
			}),
		},
		{
			name:  "Unmarshal error status (2)",
			code:  http.StatusOK,
			input: []byte(`{"error": {"code": 500, "message": "test error"}}`),
			wantGrpcError: grpcStatus.ErrorProto(&status.Status{
				Code:    http.StatusInternalServerError,
				Message: "test error",
			}),
		},
		{
			name:    "Invalid JSON",
			code:    http.StatusOK,
			input:   []byte(`invalid json`),
			wantErr: true,
		},
		{
			name:  "Empty result and error fields",
			code:  http.StatusOK,
			input: []byte(`{}`),
			want:  &structpb.Struct{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v structpb.Value
			err := pb.Unmarshal(tt.code, bytes.NewReader(tt.input), &v)
			if tt.wantGrpcError != nil {
				require.ErrorIs(t, err, tt.wantGrpcError)
				return
			}
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			test.CheckProtobufs(t, tt.want, &v, test.RequireToCheckFunc(require.Equal))
		})
	}
}
