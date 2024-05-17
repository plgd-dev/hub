package test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	wotTD "github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

func CmpThingDescription(t *testing.T, expected, got *wotTD.ThingDescription) {
	sort.Strings(expected.Type.StringArray)
	sort.Strings(got.Type.StringArray)
	sortProperties := func(td *wotTD.ThingDescription) {
		for key, prop := range td.Properties {
			for idx, form := range prop.Forms {
				if form.Op.StringArray == nil {
					continue
				}
				sort.Strings(form.Op.StringArray)
				prop.Forms[idx] = form
			}
			if prop.Type == nil {
				continue
			}
			sort.Strings(prop.Type.StringArray)
			td.Properties[key] = prop
		}
	}
	sortProperties(expected)
	sortProperties(got)
	require.Equal(t, expected, got)
}
