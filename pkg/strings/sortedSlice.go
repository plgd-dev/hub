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

// Get elements of the first slice not contained in the second
func (slice SortedSlice) Difference(second SortedSlice) SortedSlice {
	var diff SortedSlice
	var j int
	for i := range slice {
		for (j < len(second)) && (second[j] < slice[i]) {
			j++
		}
		if j == len(second) {
			diff = append(diff, slice[i:]...)
			break
		}
		if second[j] != slice[i] {
			diff = append(diff, slice[i])
		}
	}

	return diff
}

func (slice SortedSlice) Insert(elems ...string) SortedSlice {
	for _, v := range elems {
		i := sort.SearchStrings([]string(slice), v)
		if slice.Contains(v) {
			continue
		}
		slice = append(slice, "")
		copy(slice[i+1:], slice[i:])
		slice[i] = v
	}
	return slice
}

func (slice SortedSlice) Contains(s string) bool {
	i := sort.SearchStrings([]string(slice), s)
	return i < len(slice) && slice[i] == s
}

func (slice SortedSlice) Remove(elems ...string) SortedSlice {
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
