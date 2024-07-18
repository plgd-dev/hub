package pb

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"google.golang.org/protobuf/encoding/protojson"
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

func (x *Token) ToMap() (map[string]interface{}, error) {
	v := protojson.MarshalOptions{
		AllowPartial:    true,
		EmitUnpopulated: true,
	}
	data, err := v.Marshal(x)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func replaceStrToInt64(m map[string]interface{}, keys ...string) error {
	var errs *multierror.Error
	for _, k := range keys {
		exp, ok := m[k]
		if ok {
			str, ok := exp.(string)
			if ok {
				i, err := strconv.ParseInt(str, 10, 64)
				if err != nil {
					errs = multierror.Append(errs, fmt.Errorf("cannot convert key %v to int64, %w", k, err))
				} else {
					m[k] = i
				}
			}
		}
	}
	return errs.ErrorOrNil()
}

func replaceInt64ToStr(m map[string]interface{}, keys ...string) {
	for _, k := range keys {
		exp, ok := m[k]
		if ok {
			i, ok := exp.(int64)
			if ok {
				m[k] = strconv.FormatInt(i, 10)
			}
		}
	}
}

func (x *Token) ToBsonMap() (map[string]interface{}, error) {
	m, err := x.ToMap()
	if err != nil {
		return nil, err
	}
	m["_id"] = x.GetId()
	delete(m, "id")
	err = replaceStrToInt64(m, ExpirationKey, TimestampKey)
	if err != nil {
		return nil, err
	}
	blackListed, ok := m[BlackListedKey]
	if ok {
		mapBlacklisted, ok := blackListed.(map[string]interface{})
		if ok {
			err = replaceStrToInt64(mapBlacklisted, TimestampKey)
			if err != nil {
				return nil, err
			}
		}
	}
	return m, nil
}

func (x *Token) FromMap(m map[string]interface{}) error {
	if x == nil {
		return errTokenIsNil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	v := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}
	return v.Unmarshal(data, x)
}

func (x *Token) FromBsonMap(m map[string]interface{}) error {
	if x == nil {
		return errTokenIsNil
	}
	m["id"] = m["_id"]
	delete(m, "_id")

	replaceInt64ToStr(m, ExpirationKey, TimestampKey)
	blackListed, ok := m[BlackListedKey]
	if ok {
		mapBlacklisted, ok := blackListed.(map[string]interface{})
		if ok {
			replaceInt64ToStr(mapBlacklisted, TimestampKey)
		}
	}

	return x.FromMap(m)
}

func (x *Token_BlackListed) ToMap() (map[string]interface{}, error) {
	v := protojson.MarshalOptions{
		AllowPartial:    true,
		EmitUnpopulated: true,
	}
	data, err := v.Marshal(x)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (x *Token_BlackListed) FromMap(m map[string]interface{}) error {
	if x == nil {
		return errors.New("Token_BlackListed is nil")
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	v := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}
	return v.Unmarshal(data, x)
}

func (x *Token_BlackListed) ToBsonMap() (map[string]interface{}, error) {
	m, err := x.ToMap()
	if err != nil {
		return nil, err
	}
	err = replaceStrToInt64(m, TimestampKey)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (x *Token_BlackListed) FromBsonMap(m map[string]interface{}) error {
	if x == nil {
		return errors.New("Token_BlackListed is nil")
	}
	replaceInt64ToStr(m, TimestampKey)
	return x.FromMap(m)
}
