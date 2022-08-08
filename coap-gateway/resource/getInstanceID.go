package resource

import "hash/crc32"

func GetInstanceID(href string) int64 {
	h := crc32.New(crc32.IEEETable)
	h.Write([]byte(href))
	return int64(h.Sum32())
}
