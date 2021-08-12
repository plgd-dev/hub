package strings

// Return slice of unique elements contained by both input slices
func Intersection(s1, s2 []string) []string {
	hash := make(map[string]bool)
	for _, e := range s1 {
		// false => value exists but hasn't been appended to result slice
		hash[e] = false
	}
	var inter []string
	for _, e := range s2 {
		if v, ok := hash[e]; ok {
			if !v {
				inter = append(inter, e)
				// true => has been appended to result slice, don't append it again
				hash[e] = true
			}
		}
	}
	return inter
}
