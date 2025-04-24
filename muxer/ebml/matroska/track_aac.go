package matroska

import (
	"time"

	"github.com/greendrake/cctv/muxer/ebml/core"
	"github.com/greendrake/cctv/muxer/ebml/webm"
)

type trackAAC struct {
	UnimplementedTrack
}

func NewTrackAAC(samplingFrequency int, channels int) Track {
	t := &trackAAC{
		UnimplementedTrack: UnimplementedTrack{
			track: webm.TrackEntry{
				Name:        "Audio(AAC)",
				TrackNumber: 1,
				TrackUID:    getTrackUID(),
				CodecID:     core.AudioCodecAAC,
				TrackType:   core.TrackTypeAudio,
				Audio: &webm.Audio{
					SamplingFrequency: float64(samplingFrequency),
					Channels:          uint64(channels),
				},
			},
		},
	}

	return t
}

func (tis *trackAAC) Write(timestamp time.Duration, b []byte, keyframe ...bool) (int, error) {
	return tis.UnimplementedTrack.Write(timestamp, b, true)
}
