package service_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/genproto/googleapis/rpc/status"
	grpcStatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Decoder = interface {
	Decode(interface{}) error
}

func UnmarshalError(data []byte) error {
	var s status.Status
	err := protojson.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	return grpcStatus.ErrorProto(&s)
}

func Unmarshal(code int, input io.Reader, v protoreflect.ProtoMessage) error {
	var data json.RawMessage
	err := json.NewDecoder(input).Decode(&data)

	if err != nil {
		return err
	}
	fmt.Printf("data: %s\n", data)

	if code != http.StatusOK {
		return UnmarshalError(data)
	}

	var item struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}

	err = jsoniter.Unmarshal(data, &item)
	if err != nil {
		return err
	}
	if len(item.Result) == 0 && len(item.Error) == 0 {
		u := protojson.UnmarshalOptions{
			DiscardUnknown: true,
		}
		err := u.Unmarshal(data, v)
		if err != nil {
			return err
		}
		return nil
	}
	if len(item.Error) > 0 {
		return UnmarshalError(item.Error)
	}
	u := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	err = u.Unmarshal(item.Result, v)
	if err != nil {
		return err
	}
	return nil
}
