package pb

import (
	"cmp"
	"slices"
	"strconv"
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
	// if the first filter is All, we can ignore all other filters
	first := idfilter[0]
	if first.GetId() == "" && first.GetAll() {
		return nil
	}
	return idfilter
}

// Normalizing the IDFilter entails:
// - ordering the filters:
//   - primarily by ID
//   - secondarily by type (All, Latest, Value)
//   - tertiary for Value type by value
//
// - removing duplicates
// - removing filters that are redundant due to other filters (eg if All is specified, no other filters are needed)
func NormalizeIdFilter(idfilter []*IDFilter) []*IDFilter {
	slices.SortFunc(idfilter, compareIdFilter)
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
	// document IDs for which we want full documents
	All []string
	// document IDs for which we want the latest version
	Latest []string
	// map of document IDs and specific versions we want
	Versions map[string][]uint64
}

func (vf *VersionFilter) IsEmpty() bool {
	return len(vf.All) == 0 && len(vf.Latest) == 0 && len(vf.Versions) == 0
}

func PartitionIDFilter(idfilter []*IDFilter) VersionFilter {
	idFilter := NormalizeIdFilter(idfilter)
	if len(idFilter) == 0 {
		return VersionFilter{}
	}

	vf := VersionFilter{
		Versions: make(map[string][]uint64),
	}

	// empty ID ("") is interpreted as filtering by ID disabled, therefore we return all IDs
	// if we requested latest for all ids then we can skip specific IDs requesting latest
	hasAllIdsLatest := false
	// if we requested a specific version for all IDs then we can skip specific IDs requesting that version
	hasAllIdsVersion := func(version uint64) bool {
		allVersions, ok := vf.Versions[""]
		return ok && slices.Contains(allVersions, version)
	}

	for _, idf := range idFilter {
		if idf.GetAll() {
			vf.All = append(vf.All, idf.GetId())
			continue
		}

		if idf.GetLatest() {
			if hasAllIdsLatest {
				continue
			}
			if idf.GetId() == "" {
				hasAllIdsLatest = true
			}
			vf.Latest = append(vf.Latest, idf.GetId())
			continue
		}

		version := idf.GetValue()
		if hasAllIdsVersion(version) {
			continue
		}
		idVersions := vf.Versions[idf.GetId()]
		idVersions = append(idVersions, version)
		vf.Versions[idf.GetId()] = idVersions
	}
	return vf
}

func parseVersion(v string) isIDFilter_Version {
	switch v {
	case "", "all":
		return &IDFilter_All{
			All: true,
		}
	case "latest":
		return &IDFilter_Latest{
			Latest: true,
		}
	default:
		ver, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil
		}
		return &IDFilter_Value{
			Value: ver,
		}
	}
}

// we are permissive in parsing id filter
func idFilterFromString(v string) *IDFilter {
	if len(v) == 0 {
		return nil
	}
	for len(v) > 0 && v[0] == '/' {
		v = v[1:]
	}
	idHref := strings.SplitN(v, "/", 2)
	if len(idHref) < 2 {
		ver := parseVersion(v)
		if ver != nil {
			return &IDFilter{
				Version: ver,
			}
		}
		return &IDFilter{
			Id: v,
			Version: &IDFilter_All{
				All: true,
			},
		}
	}

	ver := parseVersion(idHref[1])
	if ver == nil {
		return nil
	}
	return &IDFilter{
		Id:      idHref[0],
		Version: ver,
	}
}

func IDFilterFromString(filter []string) []*IDFilter {
	if len(filter) == 0 {
		return nil
	}
	ret := make([]*IDFilter, 0, len(filter))
	for _, s := range filter {
		f := idFilterFromString(s)
		if f == nil {
			continue
		}
		ret = append(ret, f)
	}
	return ret
}

func (r *GetConditionsRequest) ConvertHTTPIDFilter() []*IDFilter {
	return IDFilterFromString(r.GetHttpIdFilter())
}

func (r *GetConfigurationsRequest) ConvertHTTPIDFilter() []*IDFilter {
	return IDFilterFromString(r.GetHttpIdFilter())
}

func (r *GetAppliedConfigurationsRequest) ConvertHTTPConfigurationIdFilter() []*IDFilter {
	return IDFilterFromString(r.GetHttpConfigurationIdFilter())
}

func (r *GetAppliedConfigurationsRequest) ConvertHTTPConditionIdFilter() []*IDFilter {
	return IDFilterFromString(r.GetHttpConditionIdFilter())
}

func (r *DeleteConfigurationsRequest) ConvertHTTPIDFilter() []*IDFilter {
	return IDFilterFromString(r.GetHttpIdFilter())
}

func (r *DeleteConditionsRequest) ConvertHTTPIDFilter() []*IDFilter {
	return IDFilterFromString(r.GetHttpIdFilter())
}
