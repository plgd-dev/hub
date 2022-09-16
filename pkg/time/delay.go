package time

import (
	"math/rand"
	"time"
)

func GetRandomDelayGenerator(interval time.Duration) func() time.Duration {
	if interval <= 0 {
		interval = time.Second * 5
	}
	return func() time.Duration {
		return time.Duration(rand.Int63n(int64(interval)))
	}
}
