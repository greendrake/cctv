package frame

import (
	"time"
)

const (
	ExpectedAudioSampleRate uint16 = 8000
)

type Meta struct {
	Length     uint32
	Width      uint16
	Height     uint16
	FPS        uint8
	Type       Type
	MediaType  string
	DateTime   time.Time     // Full date time. Key frames only
	Timestamp  time.Duration // starts from 0 when the stream starts
	SampleRate uint16
}

type Frame struct {
	Meta *Meta
	Data *[]byte
}

func (f Frame) IsVideoKeyFrame() bool {
	return f.Meta.Type == T_VideoI
}

func (f Frame) IsVideo() bool {
	return f.IsVideoKeyFrame() || f.Meta.Type == T_VideoP
}

func (f Frame) IsAudio() bool {
	return f.Meta.Type == T_Audio
}

type RawFrame struct {
	Type   Type
	Header Header
	Meta   *Meta
	Data   *[]byte
}

func (f RawFrame) GetMeta() *Meta {
	if f.Meta == nil {
		f.Meta = &Meta{
			Type:      f.Type,
			MediaType: f.Header.GetMediaType(f.Type),
			Length:    f.Header.GetLength(),
		}
		if f.Type == T_VideoI || f.Type == T_Picture {
			f.Meta.DateTime = f.Header.(DateTimeHolder).GetDateTime()
		}
		if f.Type == T_VideoI {
			hi := f.Header.(*HeaderVideoI)
			f.Meta.Width = hi.GetWidth()
			f.Meta.Height = hi.GetHeight()
			f.Meta.FPS = hi.GetFPS()
		}
		if f.Type == T_Audio {
			s := f.Header.(*HeaderAudio).SampleRate
			if s == 2 {
				f.Meta.SampleRate = ExpectedAudioSampleRate
			}
		}
	}
	return f.Meta
}
