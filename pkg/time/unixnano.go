package time

import "time"

// MaxTime = 292277024627-12-06 15:30:07.999999999 +0000 UTC
var MaxTime = time.Unix(1<<63-62135596801, 999999999)

// MinTime = 1970-01-01 00:00:00 +0000 UTC
var MinTime = time.Unix(0, 0)

func UnixNano(t time.Time) int64 {
	v := int64(0)
	if !t.IsZero() {
		v = t.UnixNano()
	}
	return v
}

func UnixSec(t time.Time) int64 {
	v := int64(0)
	if !t.IsZero() {
		v = t.Unix()
	}
	return v
}

func Unix(sec int64, nsec int64) time.Time {
	if sec != 0 || nsec != 0 {
		return time.Unix(sec, nsec)
	}
	return time.Time{}
}
