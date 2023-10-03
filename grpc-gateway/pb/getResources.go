package pb

import (
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

const etagQueryKey = "etag"

// we are permissive in parsing resource id filter
func resourceIdFilterFromString(v string) *ResourceIdFilter {
	if len(v) == 0 {
		return nil
	}
	if v[0] == '/' {
		v = v[1:]
	}
	deviceIDHref := strings.SplitN(v, "/", 2)
	if len(deviceIDHref) != 2 {
		return &ResourceIdFilter{
			ResourceId: &commands.ResourceId{
				DeviceId: v,
			},
		}
	}

	hrefQuery := strings.SplitN(deviceIDHref[1], "?", 2)
	if len(hrefQuery) < 2 {
		return &ResourceIdFilter{
			ResourceId: &commands.ResourceId{
				DeviceId: deviceIDHref[0],
				Href:     "/" + deviceIDHref[1],
			},
		}
	}
	values, err := url.ParseQuery(hrefQuery[1])
	if err != nil {
		return &ResourceIdFilter{
			ResourceId: &commands.ResourceId{
				DeviceId: deviceIDHref[0],
				Href:     "/" + hrefQuery[0],
			},
		}
	}
	etags := make([][]byte, 0, len(values[etagQueryKey]))
	for _, etag := range values[etagQueryKey] {
		e, err := base64.StdEncoding.DecodeString(etag)
		if err == nil {
			etags = append(etags, e)
		}
	}
	if len(etags) == 0 {
		etags = nil
	}
	return &ResourceIdFilter{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceIDHref[0],
			Href:     "/" + hrefQuery[0],
		},
		Etag: etags,
	}
}

func ResourceIdFilterFromString(filter []string) []*ResourceIdFilter {
	if len(filter) == 0 {
		return nil
	}
	ret := make([]*ResourceIdFilter, 0, len(filter))
	for _, s := range filter {
		f := resourceIdFilterFromString(s)
		if f == nil {
			continue
		}
		ret = append(ret, f)
	}
	return ret
}

func (r *GetResourcesRequest) ConvertHTTPResourceIDFilter() []*ResourceIdFilter {
	return ResourceIdFilterFromString(r.GetHttpResourceIdFilter())
}
