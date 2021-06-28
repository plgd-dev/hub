package service

import (
	"encoding/base64"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"
	"google.golang.org/protobuf/encoding/protojson"
)

type jsonMarshaler struct {
	*runtime.JSONPb
}

func newJsonMarshaler() *jsonMarshaler {
	return &jsonMarshaler{
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
func (*jsonMarshaler) ContentType(_ interface{}) string {
	return "application/json"
}

func replaceContent(val map[interface{}]interface{}) (interface{}, bool) {
	contentTypeI, ok := val["contentType"]
	if !ok {
		return nil, false
	}
	contentType, ok := contentTypeI.(string)
	if !ok {
		return nil, false
	}
	dataI, ok := val["data"]
	if !ok {
		return nil, false
	}
	datab64, ok := dataI.(string)
	if !ok {
		return nil, false
	}
	data, err := base64.StdEncoding.DecodeString(datab64)
	if err != nil {
		return nil, false
	}
	switch contentType {
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		var v interface{}
		err := cbor.Decode(data, &v)
		if err != nil {
			return nil, false
		}
		return v, true
	case message.AppJSON.String():
		var v interface{}
		err := json.Decode(data, &v)
		if err != nil {
			return nil, false
		}
		return v, true
	case message.TextPlain.String():
		return string(data), true
	}
	return nil, false
}

func modify(v interface{}) (interface{}, bool) {
	valMap, ok := v.(map[interface{}]interface{})
	if ok {
		newContent, replace := replaceContent(valMap)
		if replace {
			return newContent, replace
		}
		for key, v := range valMap {
			newContent, replace := modify(v)
			if replace {
				valMap[key] = newContent
			}
		}
	}
	valArr, ok := v.([]interface{})
	if ok {
		for _, v := range valArr {
			modify(v)
		}
	}
	return nil, false
}

// Marshal marshals "v" into JSON.
func (j *jsonMarshaler) Marshal(v interface{}) ([]byte, error) {
	data, err := j.JSONPb.Marshal(v)
	if err != nil {
		return data, err
	}
	var val interface{}
	err = json.Decode(data, &val)
	if err != nil {
		return data, nil
	}
	newContent, replace := modify(val)
	if replace {
		val = newContent
	}
	newData, err := json.Encode(val)
	if err != nil {
		return data, nil
	}
	fmt.Printf("newData: %s\n", newData)

	return newData, err
}
