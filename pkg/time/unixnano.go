package time

import "time"

func UnixNano(t time.Time) int64 {
	v := int64(0)
	if !t.IsZero() {
		v = t.UnixNano()
	}
	return v
}

func Unix(sec int64, nsec int64) time.Time {
	if sec != 0 || nsec != 0 {
		return time.Unix(sec, nsec)
	}
	return time.Time{}
}
