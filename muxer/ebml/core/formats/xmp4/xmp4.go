package xmp4

import (
	"fmt"
	"io"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/greendrake/cctv/muxer/ebml/core/codecs"
	"github.com/pion/rtp"

	mp4go2rtc "github.com/AlexxIT/go2rtc/pkg/mp4"
)

type Muxer struct {
	w     io.Writer
	muxer mp4go2rtc.Muxer

	codecs     []*core.Codec
	videoIndex int
	audioIndex int
}

// NewMuxer mp4 muxer, from github.com/AlexxIT/go2rtc/pkg/mp4
func NewMuxer(w io.Writer) *Muxer {
	return &Muxer{
		w:          w,
		codecs:     nil,
		videoIndex: -1,
		audioIndex: -1,
	}
}

func (m *Muxer) AddTrack(codecs ...*core.Codec) error {
	data, err := m.muxer.GetInit(codecs)
	if err != nil {
		return err
	}

	if _, err := m.w.Write(data); err != nil {
		return err
	}

	m.codecs = append(m.codecs, codecs...)

	for idx, codec := range m.codecs {
		switch codec.Name {
		case core.CodecH264, core.CodecH265, core.CodecAV1, core.CodecVP8, core.CodecVP9:
			m.videoIndex = idx
		case core.CodecPCMA, core.CodecPCMU, core.CodecAAC, core.CodecOpus, core.CodecMP3:
			m.audioIndex = idx
		}
	}

	return nil
}

func (m *Muxer) getCodec(index uint8) (uint8, bool) {
	if index < 0 {
		return 0, false
	}

	switch int(index) {
	case m.videoIndex:
		return index, true
	case m.audioIndex:
		return index, true
	}

	return 0, false
}

func (m *Muxer) WriteVideo(timestamp time.Duration, data []byte) error {
	if m.videoIndex == -1 {
		return fmt.Errorf("no track")
	}

	packet := &rtp.Packet{
		Header: rtp.Header{
			Timestamp: uint32(timestamp.Milliseconds()),
		},
		Payload: data,
	}

	buff := m.muxer.Marshal(uint8(m.videoIndex), packet)
	if _, err := m.w.Write(buff); err != nil {
		return err
	}

	return nil
}

func (m *Muxer) WriteAudio(timestamp time.Duration, data []byte) error {
	if m.audioIndex == -1 {
		return fmt.Errorf("no track")
	}

	packet := &rtp.Packet{
		Header: rtp.Header{
			Timestamp: uint32(timestamp.Milliseconds()),
		},
		Payload: data,
	}

	buff := m.muxer.Marshal(uint8(m.audioIndex), packet)
	if _, err := m.w.Write(buff); err != nil {
		return err
	}

	return nil
}

func NewH264Codec(sps, pps []byte) *core.Codec {
	return &core.Codec{
		Name:        core.CodecH264,
		ClockRate:   90000,
		FmtpLine:    codecs.NewH264Param(sps, pps).GetFmtpString(),
		PayloadType: core.PayloadTypeRAW,
	}
}

func NewH265Codec(vps, sps, pps []byte) *core.Codec {
	return &core.Codec{
		Name:        core.CodecH265,
		ClockRate:   90000,
		FmtpLine:    codecs.NewH265Param(vps, sps, pps).GetFmtpString(),
		PayloadType: core.PayloadTypeRAW,
	}
}

func NewOpusCodec(channel int) *core.Codec {
	return &core.Codec{
		Name:        core.CodecOpus,
		ClockRate:   uint32(48000),
		Channels:    uint16(channel),
		PayloadType: core.PayloadTypeRAW,
	}
}

func NewAACCodec(channel int, clockRate int) *core.Codec {
	return &core.Codec{
		Name:        core.CodecAAC,
		ClockRate:   uint32(clockRate),
		Channels:    uint16(channel),
		PayloadType: core.PayloadTypeRAW,
	}
}

func NewPCMACodec(channel int, clockRate int) *core.Codec {
	return &core.Codec{
		Name:        core.CodecPCMA,
		ClockRate:   uint32(clockRate),
		Channels:    uint16(channel),
		PayloadType: core.PayloadTypeRAW,
	}
}

func NewPCMUCodec(channel int, clockRate int) *core.Codec {
	return &core.Codec{
		Name:        core.CodecPCMU,
		ClockRate:   uint32(clockRate),
		Channels:    uint16(channel),
		PayloadType: core.PayloadTypeRAW,
	}
}
