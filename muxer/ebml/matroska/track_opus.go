package matroska

import (
	"time"

	"github.com/greendrake/cctv/muxer/ebml/core"
	"github.com/greendrake/cctv/muxer/ebml/webm"
)

type trackOpus struct {
	UnimplementedTrack
}

func NewTrackOpus(channels int) Track {
	t := &trackOpus{
		UnimplementedTrack: UnimplementedTrack{
			track: webm.TrackEntry{
				Name:         "Audio(opus)",
				TrackNumber:  1,
				TrackUID:     getTrackUID(),
				CodecID:      core.AudioCodecOPUS,
				TrackType:    core.TrackTypeAudio,
				CodecPrivate: nil,
				Audio: &webm.Audio{
					SamplingFrequency: float64(48000),
					Channels:          uint64(channels),
				},
			},
		},
	}

	return t
}

func (tis *trackOpus) Write(timestamp time.Duration, b []byte, keyframe ...bool) (int, error) {
	return tis.UnimplementedTrack.Write(timestamp, b, true)
}
