package service

import "hash/crc32"

func getInstanceID(href string) int64 {
	h := crc32.New(crc32.IEEETable)
	h.Write([]byte(href))
	return int64(h.Sum32())
}
