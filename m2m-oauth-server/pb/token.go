package pb

import (
	"errors"
	"fmt"

	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
)

var errTokenIsNil = errors.New("Token is nil")

func (x *Token) Validate() error {
	if x == nil {
		return errTokenIsNil
	}
	if x.GetId() == "" {
		return errors.New("Token.Id is empty")
	}
	if x.GetOwner() == "" {
		return errors.New("Token.Owner is empty")
	}
	if x.GetClientId() == "" {
		return errors.New("Token.ClientId is empty")
	}
	if x.GetIssuedAt() == 0 {
		return errors.New("Token.Timestamp is empty")
	}
	return nil
}

func (x *Token) jsonToBSONTag(json map[string]interface{}) error {
	json["_id"] = x.GetId()
	delete(json, "id")
	if _, err := pkgMongo.ConvertStringValueToInt64(json, false, "."+IssuedAtKey); err != nil {
		return fmt.Errorf("cannot convert issueAt to int64: %w", err)
	}
	if _, err := pkgMongo.ConvertStringValueToInt64(json, true, "."+ExpirationKey); err != nil {
		return fmt.Errorf("cannot convert expiration to int64: %w", err)
	}
	if _, err := pkgMongo.ConvertStringValueToInt64(json, true, "."+BlackListedKey+"."+TimestampKey); err != nil {
		return fmt.Errorf("cannot convert blacklisted.timestamp to int64: %w", err)
	}
	return nil
}

func (x *Token) MarshalBSON() ([]byte, error) {
	if x == nil {
		return nil, errTokenIsNil
	}
	return pkgMongo.MarshalProtoBSON(x, x.jsonToBSONTag)
}

func (x *Token) UnmarshalBSON(data []byte) error {
	if x == nil {
		return errTokenIsNil
	}
	var id string
	update := func(json map[string]interface{}) error {
		idI, ok := json["_id"]
		if ok {
			id = idI.(string)
		}
		delete(json, "_id")
		return nil
	}
	err := pkgMongo.UnmarshalProtoBSON(data, x, update)
	if err != nil {
		return err
	}
	if x.GetId() == "" && id != "" {
		x.Id = id
	}
	return nil
}

func (x *Token_BlackListed) jsonToBSONTag(json map[string]interface{}) error {
	if _, err := pkgMongo.ConvertStringValueToInt64(json, false, "."+TimestampKey); err != nil {
		return fmt.Errorf("cannot convert timestamp to int64: %w", err)
	}
	return nil
}

func (x *Token_BlackListed) MarshalBSON() ([]byte, error) {
	return pkgMongo.MarshalProtoBSON(x, x.jsonToBSONTag)
}
