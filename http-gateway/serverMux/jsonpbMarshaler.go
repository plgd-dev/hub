package serverMux

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

type JsonpbMarshaler struct {
	*runtime.JSONPb
}

// NewJsonpbMarshaler is a proto marshaler that uses jsonpb. proto <=> jsonpb
func NewJsonpbMarshaler() *JsonpbMarshaler {
	return &JsonpbMarshaler{
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
func (*JsonpbMarshaler) ContentType(_ interface{}) string {
	return "application/jsonpb"
}
