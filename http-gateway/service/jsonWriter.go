package service

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/gogo/protobuf/proto"
)

const contentTypeHeaderKey = "Content-Type"

func structToMap(item interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	if item == nil {
		return res
	}
	v := reflect.TypeOf(item)
	reflectValue := reflect.ValueOf(item)
	reflectValue = reflect.Indirect(reflectValue)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		tag := v.Field(i).Tag.Get("json")
		field := reflectValue.Field(i).Interface()
		var keyName string
		switch tag {
		case "-":
		case "":
			keyName = v.Field(i).Name
		default:
			tags := strings.Split(tag, ",")
			keyName = tags[0]
		}
		if keyName != "" {
			if v.Field(i).Type.Kind() == reflect.Struct {
				res[keyName] = structToMap(field)
			} else {
				res[keyName] = field
			}
		}
	}
	return res
}

func jsonResponseWriter(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		return nil
	}
	w.Header().Set(contentTypeHeaderKey, message.AppJSON.String())
	if pb, ok := v.(proto.Message); ok {
		v = structToMap(pb)
	}

	return json.WriteTo(w, v)
}
