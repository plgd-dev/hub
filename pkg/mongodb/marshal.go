package mongodb

import (
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func UnmarshalProtoBSON(data []byte, m proto.Message) error {
	var obj map[string]interface{}
	if err := bson.Unmarshal(data, &obj); err != nil {
		return err
	}
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonData, m)
}

func MarshalProtoBSON(m proto.Message) ([]byte, error) {
	data, err := protojson.Marshal(m)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}
	return bson.Marshal(obj)
}
