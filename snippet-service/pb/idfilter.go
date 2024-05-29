package pb

import (
	"cmp"
	"slices"
	"strings"
)

func compareIdFilter(i, j *IDFilter) int {
	// compare by Id first
	if i.GetId() != j.GetId() {
		return strings.Compare(i.GetId(), j.GetId())
	}
	// then by type
	// All is always first
	if i.GetAll() {
		if j.GetAll() {
			return 0
		}
		return -1
	}
	// Latest is always second
	if i.GetLatest() {
		if j.GetAll() {
			return 1
		}
		if j.GetLatest() {
			return 0
		}
		return -1
	}
	// Values are always last, ordered by stored value
	if j.GetAll() || j.GetLatest() {
		return 1
	}
	return cmp.Compare(i.GetValue(), j.GetValue())
}

func checkEmptyIdFilter(idfilter []*IDFilter) []*IDFilter {
	// if an empty query is provided, return all
	if len(idfilter) == 0 {
		return nil
	}
	slices.SortFunc(idfilter, compareIdFilter)
	// if the first filter is All, we can ignore all other filters
	first := idfilter[0]
	if first.GetId() == "" && first.GetAll() {
		return nil
	}
	return idfilter
}

func NormalizeIdFilter(idfilter []*IDFilter) []*IDFilter {
	idfilter = checkEmptyIdFilter(idfilter)
	if len(idfilter) == 0 {
		return nil
	}

	updatedFilter := make([]*IDFilter, 0)
	var idAll bool
	var idLatest bool
	var idValue bool
	var idValueVersion uint64
	setNextLatest := func(idf *IDFilter) {
		// we already have the latest filter
		if idLatest {
			// skip
			return
		}
		idLatest = true
		updatedFilter = append(updatedFilter, idf)
	}
	setNextValue := func(idf *IDFilter) {
		value := idf.GetValue()
		if idValue && value == idValueVersion {
			// skip
			return
		}
		idValue = true
		idValueVersion = value
		updatedFilter = append(updatedFilter, idf)
	}
	prevID := ""
	for _, idf := range idfilter {
		if idf.GetId() != prevID {
			idAll = idf.GetAll()
			idLatest = idf.GetLatest()
			idValue = !idAll && !idLatest
			idValueVersion = idf.GetValue()
			updatedFilter = append(updatedFilter, idf)
		}

		if idAll {
			goto next
		}

		if idf.GetLatest() {
			setNextLatest(idf)
			goto next
		}

		setNextValue(idf)

	next:
		prevID = idf.GetId()
	}
	return updatedFilter
}

type VersionFilter struct {
	latest   bool
	versions []uint64
}

func (vf *VersionFilter) Latest() bool {
	return vf.latest
}

func (vf *VersionFilter) Versions() []uint64 {
	return vf.versions
}

func PartitionIDFilter(idfilter []*IDFilter) ([]string, map[string]VersionFilter) {
	idFilter := NormalizeIdFilter(idfilter)
	if len(idFilter) == 0 {
		return nil, nil
	}
	idVersionAll := make([]string, 0)
	idVersions := make(map[string]VersionFilter, 0)
	hasAllIdsLatest := func() bool {
		vf, ok := idVersions[""]
		return ok && vf.latest
	}
	hasAllIdsVersion := func(version uint64) bool {
		vf, ok := idVersions[""]
		return ok && slices.Contains(vf.versions, version)
	}
	for _, idf := range idFilter {
		if idf.GetAll() {
			idVersionAll = append(idVersionAll, idf.GetId())
			continue
		}
		vf := idVersions[idf.GetId()]
		if idf.GetLatest() {
			if hasAllIdsLatest() {
				continue
			}
			vf.latest = true
			idVersions[idf.GetId()] = vf
			continue
		}
		version := idf.GetValue()
		if hasAllIdsVersion(version) {
			continue
		}
		vf.versions = append(vf.versions, version)
		idVersions[idf.GetId()] = vf
	}
	return idVersionAll, idVersions
}
