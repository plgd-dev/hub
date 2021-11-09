package test

import (
	"log"
	"reflect"
)

// Get json tag of field with given name in struct v
func FieldJsonTag(v interface{}, fieldName string) string {
	field, ok := reflect.TypeOf(v).FieldByName(fieldName)
	if !ok {
		log.Fatalf("invalid fieldName %v", fieldName)
	}
	return field.Tag.Get("json")
}
