package service

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func checkTimeToLive(timeToLive int64) error {
	if timeToLive != 0 && timeToLive < int64(time.Millisecond*100) {
		return status.Errorf(codes.InvalidArgument, "timeToLive(`%v`) is less than 100ms", time.Duration(timeToLive))
	}
	return nil
}

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
