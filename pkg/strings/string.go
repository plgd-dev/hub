package strings

func ToString(v interface{}) (string, bool) {
	if v == nil {
		return "", false
	}
	val, ok := v.(string)
	return val, ok
}
