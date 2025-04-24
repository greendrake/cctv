package matroska

import (
	"bytes"
	"encoding/binary"
	"time"

	"gitee.com/general252/go-wav"
	"github.com/greendrake/cctv/muxer/ebml/core"
	"github.com/greendrake/cctv/muxer/ebml/webm"
)

type trackPCMA struct {
	UnimplementedTrack
}

func NewTrackPCMA(sampleRate int, channels int) Track {
	wavFormat := wav.WavFormat{
		AudioFormat:   wav.AudioFormatALaw,
		NumChannels:   uint16(channels),
		SampleRate:    uint32(sampleRate),
		ByteRate:      0,
		BlockAlign:    1,
		BitsPerSample: 8,
	}

	var codecPrivate bytes.Buffer
	_ = binary.Write(&codecPrivate, binary.LittleEndian, wavFormat)
	codecPrivate.Write([]byte{0, 0})

	t := &trackPCMA{
		UnimplementedTrack: UnimplementedTrack{
			track: webm.TrackEntry{
				Name:         "Audio(pcma)",
				TrackNumber:  1,
				TrackUID:     getTrackUID(),
				CodecID:      core.AudioCodecMSACM,
				TrackType:    core.TrackTypeAudio,
				CodecPrivate: codecPrivate.Bytes(),
				Audio: &webm.Audio{
					SamplingFrequency: float64(wavFormat.SampleRate),
					Channels:          uint64(wavFormat.NumChannels),
				},
			},
		},
	}

	return t
}

func (tis *trackPCMA) Write(timestamp time.Duration, b []byte, keyframe ...bool) (int, error) {
	return tis.UnimplementedTrack.Write(timestamp, b, true)
}
