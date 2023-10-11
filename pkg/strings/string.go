package strings

import "github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"

func ToString(v interface{}) (string, bool) {
	if v == nil {
		return "", false
	}
	val, ok := v.(string)
	return val, ok
}

func ToStringArray(v interface{}) ([]string, bool) {
	switch val := v.(type) {
	case []string:
		return val, true
	case urischeme.URIScheme:
		return []string{string(val)}, true
	case []urischeme.URIScheme:
		var result []string
		for _, v := range val {
			result = append(result, string(v))
		}
		return result, true
	case []interface{}:
		var result []string
		for _, v := range val {
			if s, ok := ToString(v); ok {
				result = append(result, s)
			}
		}
		if len(val) != len(result) {
			return nil, false
		}
		return result, true
	case string:
		return []string{val}, true
	case nil:
		return nil, true
	}
	return nil, false
}
