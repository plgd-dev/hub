package service

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	ApplicationSubscribeToEventsProtoJsonContentType = "application/subscribetoeventsprotojson"
	ApplicationSubscribeToEventsMIMEWildcard         = "application/subscribetoeventswc"
)

type subscribeToEventsMarshaler struct {
	runtime.Marshaler
}

// modifyResourceIdFilter for backward compatibility we need to support resourceIdFilter as []string
func modifyResourceIdFilter(data []byte) []byte {
	resourceIdFilter0 := gjson.Get(string(data), "createSubscription.resourceIdFilter.0")
	if !resourceIdFilter0.Exists() || resourceIdFilter0.Type != gjson.String {
		return data
	}
	resourceIdFilter := gjson.Get(string(data), "createSubscription.resourceIdFilter")
	newData := string(data)
	// append resourceIdFilter to httpResourceIdFilter
	resourceIdFilter.ForEach(func(key, value gjson.Result) bool {
		newData, _ = sjson.Set(newData, "createSubscription.httpResourceIdFilter.-1", value.Str)
		return true
	})
	newData, _ = sjson.Delete(newData, "createSubscription.resourceIdFilter")
	return []byte(newData)
}

// Unmarshal unmarshals JSON "data" into "v"
func (j *subscribeToEventsMarshaler) Unmarshal(data []byte, v interface{}) error {
	data = modifyResourceIdFilter(data)
	return j.Marshaler.Unmarshal(data, v)
}

type subscribeToEventsDecoder struct {
	jsonDecoder     *json.Decoder
	newEventDecoder func(io io.Reader) runtime.Decoder
}

func (d *subscribeToEventsDecoder) Decode(v interface{}) error {
	var c map[string]interface{}
	err := d.jsonDecoder.Decode(&c)
	if err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return d.newEventDecoder(bytes.NewReader(modifyResourceIdFilter(data))).Decode(v)
}

// NewDecoder returns a Decoder which reads JSON stream from "r".
func (j *subscribeToEventsMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return &subscribeToEventsDecoder{
		jsonDecoder:     json.NewDecoder(r),
		newEventDecoder: j.Marshaler.NewDecoder,
	}
}

func newSubscribeToEventsMarshaler(marshaler runtime.Marshaler) *subscribeToEventsMarshaler {
	return &subscribeToEventsMarshaler{
		Marshaler: marshaler,
	}
}
