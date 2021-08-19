package strings

import "sort"

// Sorted slice of strings without duplicates
type SortedSlice []string

func MakeSortedSlice(slice []string) SortedSlice {
	if len(slice) <= 1 {
		return SortedSlice(slice)
	}

	tmp := make([]string, len(slice))
	copy(tmp, slice)
	sort.Strings(tmp)

	sortedSlice := make([]string, 0, len(slice))
	for _, v := range tmp {
		if (len(sortedSlice) == 0) || (sortedSlice[len(sortedSlice)-1] != v) {
			sortedSlice = append(sortedSlice, v)
		}
	}

	return SortedSlice(sortedSlice)
}

func Insert(slice SortedSlice, elems ...string) SortedSlice {
	for _, v := range elems {
		i := sort.SearchStrings([]string(slice), v)
		if Contains(slice, v) {
			continue
		}
		slice = append(slice, "")
		copy(slice[i+1:], slice[i:])
		slice[i] = v
	}
	return slice
}

func Contains(slice SortedSlice, s string) bool {
	i := sort.SearchStrings([]string(slice), s)
	return i < len(slice) && slice[i] == s
}

func Remove(slice SortedSlice, elems ...string) SortedSlice {
	deleted := 0
	for _, v := range elems {
		i := sort.SearchStrings([]string(slice), v)
		if i >= len(slice) || slice[i] != v {
			continue
		}
		copy(slice[i:], slice[i+1:])
		deleted++
	}

	if deleted > 0 {
		slice = slice[:len(slice)-deleted]
	}
	return slice
}
