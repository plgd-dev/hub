package pb

import "sort"

type SigningRecords []*SigningRecord

func (p SigningRecords) Sort() {
	sort.Slice(p, func(i, j int) bool {
		return p[i].GetId() < p[j].GetId()
	})
}
