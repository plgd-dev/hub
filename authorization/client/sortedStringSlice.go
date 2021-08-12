package client

import "sort"

type SortedSlice []string

func MakeSortedSlice(v []string) SortedSlice {
	sort.Strings(v)
	return SortedSlice(v)
}

func Insert(ss SortedSlice, s string) SortedSlice {
	i := sort.SearchStrings([]string(ss), s)
	if i < len(ss) && ss[i] == s {
		return ss
	}
	ss = append(ss, "")
	copy(ss[i+1:], ss[i:])
	ss[i] = s
	return ss
}

func Search(ss SortedSlice, s string) int {
	return sort.SearchStrings([]string(ss), s)
}

func Remove(ss SortedSlice, s string) SortedSlice {
	i := sort.SearchStrings([]string(ss), s)
	if ss[i] != s {
		return ss
	}
	if i < len(ss)-1 {
		copy(ss[i:], ss[i+1:])
	}
	return ss[:len(ss)-1]
}
