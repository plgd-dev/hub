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

type SplitFilter func(s string) bool

// Split one slice into to based on some filter.
//
// Function returns two slices, first slice contains all elements of the input slice
// that satisfy the filter, the second slice contains the rest.
func Split(s []string, f SplitFilter) ([]string, []string) {
	var s1, s2 []string
	for _, v := range s {
		if f(v) {
			s1 = append(s1, v)
		} else {
			s2 = append(s2, v)
		}
	}

	return s1, s2
}

// Return slice of unique elements of the input slice.
//
// The function is not stable, the order of elements might change.
func Unique(s []string) []string {
	if l := len(s); l == 0 {
		return nil
	}

	set := make(map[string]struct{})
	for _, v := range s {
		set[v] = struct{}{}
	}

	keys := make([]string, len(set))
	i := 0
	for k := range set {
		keys[i] = k
		i++
	}
	return keys
}

func Contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
