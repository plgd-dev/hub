package mongodb

import (
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type updateJSON = func(map[string]interface{})

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
