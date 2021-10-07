package pb

import (
	"github.com/plgd-dev/device/schema"
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

func (d *Device) ToSchema() schema.Device {
	return schema.Device{
		ID:                    d.GetId(),
		ResourceTypes:         d.GetTypes(),
		Interfaces:            d.GetInterfaces(),
		Name:                  d.GetName(),
		ManufacturerName:      LocalizedStrings(d.GetManufacturerName()).ToSchema(),
		ModelNumber:           d.GetModelNumber(),
		ProtocolIndependentID: d.GetProtocolIndependentId(),
	}
}

func SchemaLocalizedStringToProto(s schema.LocalizedString) *LocalizedString {
	return &LocalizedString{
		Language: s.Language,
		Value:    s.Value,
	}
}

func SchemaLocalizedStringsToProto(s []schema.LocalizedString) []*LocalizedString {
	if s == nil {
		return nil
	}
	l := make([]*LocalizedString, 0, len(s))
	for _, m := range s {
		l = append(l, SchemaLocalizedStringToProto(m))
	}
	return l
}

func SchemaDeviceToProto(d *schema.Device) *Device {
	if d == nil {
		return nil
	}
	return &Device{
		Id:                    d.ID,
		Types:                 d.ResourceTypes,
		Interfaces:            d.Interfaces,
		Name:                  d.Name,
		ManufacturerName:      SchemaLocalizedStringsToProto(d.ManufacturerName),
		ModelNumber:           d.ModelNumber,
		ProtocolIndependentId: d.ProtocolIndependentID,
	}
}
