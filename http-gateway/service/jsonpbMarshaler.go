package service

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

type jsonpbMarshaler struct {
	*runtime.JSONPb
}

func newJsonpbMarshaler() *jsonpbMarshaler {
	return &jsonpbMarshaler{
		JSONPb: &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		},
	}
}

// ContentType always returns "application/json".
func (*jsonpbMarshaler) ContentType(_ interface{}) string {
	return "application/jsonpb"
}
