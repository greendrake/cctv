package util

import (
	"context"
	"time"
)

func SleepCtx(ctx context.Context, delay time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(delay):
		return true
	}
}

type WrongCredentialsError struct {
	Msg string
}

func (e *WrongCredentialsError) Error() string {
	return e.Msg
}
