package matroska

import (
	"bytes"
	"encoding/binary"
	"time"

	"gitee.com/general252/go-wav"
	"github.com/greendrake/cctv/muxer/ebml/core"
	"github.com/greendrake/cctv/muxer/ebml/webm"
)

type trackG726 struct {
	UnimplementedTrack
}

func NewTrackG726(sampleRate int, channels int) Track {
	// FFmpeg-n4.4/libavformat/matroskadec.c/"A_MS/ACM"(ff_get_wav_header)
	wavFormat := wav.WavFormat{
		AudioFormat:   wav.AudioFormatG726,
		NumChannels:   uint16(channels),
		SampleRate:    uint32(sampleRate),
		ByteRate:      4000,
		BlockAlign:    1,
		BitsPerSample: 4,
	}

	var codecPrivate bytes.Buffer
	_ = binary.Write(&codecPrivate, binary.LittleEndian, wavFormat)
	codecPrivate.Write([]byte{0, 0})

	t := &trackG726{
		UnimplementedTrack: UnimplementedTrack{
			track: webm.TrackEntry{
				Name:         "Audio(g726)",
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

func (tis *trackG726) Write(timestamp time.Duration, b []byte, keyframe ...bool) (int, error) {
	return tis.UnimplementedTrack.Write(timestamp, b, true)
}
