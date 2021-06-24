package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/genproto/googleapis/rpc/status"
	grpcStatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/runtime/protoiface"
)

type Decoder = interface {
	Decode(interface{}) error
}

func UnmarshalError(data []byte) error {
	var s status.Status
	unmarshaler := jsonpb.Unmarshaler{}
	err := unmarshaler.Unmarshal(bytes.NewReader(data), &s)
	if err != nil {
		return err
	}
	return grpcStatus.ErrorProto(&s)
}

func Unmarshal(code int, decoder Decoder, v protoiface.MessageV1) error {
	var data json.RawMessage
	err := decoder.Decode(&data)
	if err != nil {
		return err
	}

	if code != http.StatusOK {
		return UnmarshalError(data)
	}

	fmt.Printf("data %s\n", data)

	var item struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}

	err = jsoniter.Unmarshal(data, &item)
	if err != nil {
		return err
	}
	if len(item.Result) == 0 && len(item.Error) == 0 {
		unmarshaler := jsonpb.Unmarshaler{}
		err = unmarshaler.Unmarshal(bytes.NewReader(data), v)
		if err != nil {
			return err
		}
		return nil
	}
	if len(item.Error) > 0 {
		return UnmarshalError(item.Error)
	}
	unmarshaler := jsonpb.Unmarshaler{}
	err = unmarshaler.Unmarshal(bytes.NewReader(item.Result), v)
	if err != nil {
		return err
	}
	return nil
}
