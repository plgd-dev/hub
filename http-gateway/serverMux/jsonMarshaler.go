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

func modify(v interface{}) (newValue interface{}, wantReplace bool, wantDelete bool) {
	if v == nil {
		return nil, false, true
	}
	valMap, ok := v.(map[interface{}]interface{})
	if ok {
		if len(valMap) == 0 {
			return nil, false, true
		}
		newContent, replace := replaceContent(valMap)
		if replace {
			return newContent, replace, false
		}
		wantReplace = false
		for key, v := range valMap {
			newContent, replace, wantDelete := modify(v)
			if replace {
				valMap[key] = newContent
				wantReplace = true
			}
			if wantDelete {
				delete(valMap, key)
				wantReplace = true
			}
		}
		if len(valMap) == 0 {
			return nil, false, true
		}
		return valMap, wantReplace, false
	}
	valArr, ok := v.([]interface{})
	if ok {
		wantReplace := false
		for idx, v := range valArr {
			if _, _, wantDelete := modify(v); wantDelete {
				wantReplace = true
				valArr = append(valArr[:idx], valArr[idx+1:]...)
			}
		}
		if len(valArr) == 0 {
			return nil, false, true
		}
		if wantReplace {
			return valArr, true, false
		}
	}
	if val, ok := v.(string); ok && len(val) == 0 {
		return nil, false, true
	}
	return nil, false, false
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
	newContent, wantReplace, _ := modify(val)
	if wantReplace {
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
