package utils

import (
	"fmt"
	"time"
)

func Retry[T any](attempts int, sleep time.Duration, f func() (T, error)) (result T, err error) {
	for attempt := range attempts {
		if attempt > 0 {
			time.Sleep(sleep)
			sleep *= 2
		}
		result, err = f()
		if err == nil {
			return result, nil
		}
	}
	return result, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
