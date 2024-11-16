package helpers

import "context"

type RetryFunc func() error

func Retry(ctx context.Context, f RetryFunc, retryAttempts int) error {
	var err error

	for i := 0; i < retryAttempts; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err = f()
		if err == nil {
			return nil
		}
	}

	return err
}
