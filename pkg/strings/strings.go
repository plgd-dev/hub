package strings

func UniqueStable(s []string) []string {
	m := make(map[string]struct{}, len(s))
	ret := make([]string, 0, len(s))
	for _, v := range s {
		if _, ok := m[v]; ok {
			continue
		}
		m[v] = struct{}{}
		ret = append(ret, v)
	}
	return ret
}
