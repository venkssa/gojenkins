package gojenkins

import (
	"context"
	"time"
)

func retryUntilFalseOrError(ctx context.Context, d time.Duration, fn func() (bool, error)) error {
	shouldRetry, err := fn()

	for err == nil && shouldRetry {
		afterChan := time.After(d)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-afterChan:
			shouldRetry, err = fn()
		}
	}
	return err
}
