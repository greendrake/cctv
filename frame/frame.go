package frame

import (
	"time"
)

type Type uint8

type Frame struct {
	IsVideo         bool
	IsHEVC          bool
	IsVideoKeyFrame bool
	IsAudio         bool
	Duration        time.Duration
	Data            *[]byte
}
