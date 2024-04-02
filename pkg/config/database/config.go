package database

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type DBUse string

func (u DBUse) ToLower() DBUse {
	return DBUse(strings.ToLower(string(u)))
}

const (
	MongoDB DBUse = "mongoDB"
	CqlDB   DBUse = "cqlDB"
)

type DBConfig interface {
	Validate() error
}

type Config[MongoConfig DBConfig, CQLDBConfig DBConfig] struct {
	Use     DBUse       `yaml:"use" json:"use"`
	MongoDB MongoConfig `yaml:"mongoDB" json:"mongoDb"`
	CqlDB   CQLDBConfig `yaml:"cqlDB" json:"cqlDb"`
}

func (c *Config[MongoConfig, CQLDBConfig]) Validate() error {
	switch c.Use.ToLower() {
	case MongoDB.ToLower():
		if reflect.ValueOf(c.MongoDB).Kind() == reflect.Ptr && reflect.ValueOf(c.MongoDB).IsNil() {
			return errors.New("mongoDB - is empty")
		}
		if err := c.MongoDB.Validate(); err != nil {
			return fmt.Errorf("mongoDB.%w", err)
		}
		c.Use = "mongoDB"
	case CqlDB.ToLower():
		if reflect.ValueOf(c.CqlDB).Kind() == reflect.Ptr && reflect.ValueOf(c.CqlDB).IsNil() {
			return errors.New("cqlDB - is empty")
		}
		if err := c.CqlDB.Validate(); err != nil {
			return fmt.Errorf("cqlDB.%w", err)
		}
		c.Use = "cqlDB"
	default:
		return fmt.Errorf("use('%v' - only %v or %v are supported)", c.Use, MongoDB, CqlDB)
	}
	return nil
}
