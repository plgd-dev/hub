package pb

import (
	"github.com/go-ocf/sdk/schema"
)

func (l *LocalizedString) ToSchema() schema.LocalizedString {
	return schema.LocalizedString{
		Language: l.GetLanguage(),
		Value:    l.GetValue(),
	}
}

type LocalizedStrings []*LocalizedString

func (s LocalizedStrings) ToSchema() []schema.LocalizedString {
	l := make([]schema.LocalizedString, 0, len(s))
	for _, m := range s {
		l = append(l, m.ToSchema())
	}
	return l
}

func (d Device) ToSchema() schema.Device {
	return schema.Device{
		ID:               d.GetId(),
		ResourceTypes:    d.GetTypes(),
		Interfaces:       d.GetInterfaces(),
		Name:             d.GetName(),
		ManufacturerName: LocalizedStrings(d.GetManufacturerName()).ToSchema(),
		ModelNumber:      d.GetModelNumber(),
	}
}

type SchemaLocalizedString schema.LocalizedString

func (s SchemaLocalizedString) ToProto() *LocalizedString {
	return &LocalizedString{
		Language: s.Language,
		Value:    s.Value,
	}
}

type SchemaLocalizedStrings []schema.LocalizedString

func (s SchemaLocalizedStrings) ToProto() []*LocalizedString {
	if s == nil {
		return nil
	}
	l := make([]*LocalizedString, 0, len(s))
	for _, m := range s {
		l = append(l, SchemaLocalizedString(m).ToProto())
	}
	return l
}

type SchemaDevice schema.Device

func (d SchemaDevice) ToProto() Device {
	return Device{
		Id:               d.ID,
		Types:            d.ResourceTypes,
		Interfaces:       d.Interfaces,
		Name:             d.Name,
		ManufacturerName: SchemaLocalizedStrings(d.ManufacturerName).ToProto(),
		ModelNumber:      d.ModelNumber,
	}
}
