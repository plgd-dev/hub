package mongodb

import (
	"encoding/json"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type updateJSON = func(map[string]interface{})

func ConvertStringValueToInt64(json map[string]interface{}, path string) {
	pos := strings.Index(path, ".")
	if pos == -1 {
		valueI, ok := json[path]
		if !ok {
			return
		}
		valueStr, ok := valueI.(string)
		if !ok {
			return
		}
		value, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return
		}
		json[path] = value
		return
	}

	elemPath := path[:pos]
	elem, ok := json[elemPath]
	if !ok {
		return
	}
	elemArray, ok := elem.([]interface{})
	if ok {
		for i, elem := range elemArray {
			elemMap, ok2 := elem.(map[string]interface{})
			if !ok2 {
				continue
			}
			ConvertStringValueToInt64(elemMap, path[pos+1:])
			elemArray[i] = elemMap
		}
		json[elemPath] = elemArray
		return
	}
	elemMap, ok := elem.(map[string]interface{})
	if !ok {
		return
	}
	ConvertStringValueToInt64(elemMap, path[pos+1:])
	json[elemPath] = elemMap
}

func UnmarshalProtoBSON(data []byte, m proto.Message, update updateJSON) error {
	var obj map[string]interface{}
	if err := bson.Unmarshal(data, &obj); err != nil {
		return err
	}
	if update != nil {
		update(obj)
	}
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonData, m)
}

func MarshalProtoBSON(m proto.Message, update updateJSON) ([]byte, error) {
	data, err := protojson.Marshal(m)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}
	if update != nil {
		update(obj)
	}
	return bson.Marshal(obj)
}
