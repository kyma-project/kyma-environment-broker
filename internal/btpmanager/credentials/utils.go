package btpmgrcreds

import (
	"fmt"
	"time"
)

func CallWithRetry[T any](closure func() (T, error), maxAttempts int, retryDelay time.Duration) (T, error) {
	doneAttempts := 0
	for {
		result, err := closure()
		if err == nil {
			return result, nil
		}
		doneAttempts++
		if doneAttempts >= maxAttempts {
			var empty T
			return empty, fmt.Errorf("%s. (All %d retries resulted in error)", err, maxAttempts)
		}
		time.Sleep(retryDelay)
	}
}
