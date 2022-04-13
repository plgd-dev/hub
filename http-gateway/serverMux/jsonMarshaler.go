package serverMux

import (
	"bytes"
	"encoding/base64"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	jsoniter "github.com/json-iterator/go"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"google.golang.org/protobuf/encoding/protojson"
)

type JsonMarshaler struct {
	*runtime.JSONPb
}

// NewJsonMarshaler is a marshaler tries to encode internal data to jsons and cbors string as json object
func NewJsonMarshaler() *JsonMarshaler {
	return &JsonMarshaler{
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
func (*JsonMarshaler) ContentType(_ interface{}) string {
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
func (j *JsonMarshaler) Marshal(v interface{}) ([]byte, error) {
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
	w := bytes.NewBuffer(make([]byte, 0, len(data)))

	encoder := jsoniter.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	err = encoder.Encode(val)
	if err != nil {
		return data, nil
	}

	return w.Bytes(), err
}
