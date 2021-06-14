package service

import (
	"unicode"

	"github.com/iancoleman/strcase"
	jsoniter "github.com/json-iterator/go"
)

type ProtobufLowerCamelExtension struct {
	jsoniter.DummyExtension
}

func (extension *ProtobufLowerCamelExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		if unicode.IsLower(rune(binding.Field.Name()[0])) || binding.Field.Name()[0] == '_' {
			continue
		}

		_, hastag := binding.Field.Tag().Lookup("protobuf")
		if !hastag {
			continue
		}

		_, hastag = binding.Field.Tag().Lookup("json")
		if !hastag {
			continue
		}
		v := strcase.ToLowerCamel(binding.Field.Name())
		binding.ToNames = []string{v}
		binding.FromNames = []string{v}
	}
}

func Encode(v interface{}) ([]byte, error) {
	cfg := jsoniter.Config{
		EscapeHTML:                    false,
		MarshalFloatWith6Digits:       true, // will lose precession
		ObjectFieldMustBeSimpleString: true, // do not unescape object field
		SortMapKeys:                   true,
	}.Froze()
	cfg.RegisterExtension(&ProtobufLowerCamelExtension{})
	return cfg.Marshal(v)
}

func Decode(data []byte, v interface{}) error {
	cfg := jsoniter.Config{
		EscapeHTML:                    false,
		MarshalFloatWith6Digits:       true, // will lose precession
		ObjectFieldMustBeSimpleString: true, // do not unescape object field
		SortMapKeys:                   true,
	}.Froze()
	cfg.RegisterExtension(&ProtobufLowerCamelExtension{})
	return cfg.Unmarshal(data, v)
}
