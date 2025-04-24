package mp4

import (
	"encoding/base64"
	"encoding/binary"
	"github.com/greendrake/cctv/muxer/core"
	"github.com/greendrake/cctv/muxer/h265"
	"github.com/greendrake/cctv/muxer/iso"
)

type Muxer struct {
	index  uint32
	dts    []uint64
	pts    []uint32
	codecs []*core.Codec
}

func (m *Muxer) AddTrack(codec *core.Codec) {
	m.dts = append(m.dts, 0)
	m.pts = append(m.pts, 0)
	m.codecs = append(m.codecs, codec)
}

func (m *Muxer) GetInit(payload []byte, ClockRate uint32) []byte {
	codec := &core.Codec{
		Name:        core.CodecH265,
		ClockRate:   ClockRate,
		PayloadType: core.PayloadTypeRAW,
		FmtpLine:    "profile-id=1",
	}
	for {
		size := 4 + int(binary.BigEndian.Uint32(payload))
		switch h265.NALUType(payload) {
		case h265.NALUTypeVPS:
			codec.FmtpLine += ";sprop-vps=" + base64.StdEncoding.EncodeToString(payload[4:size])
		case h265.NALUTypeSPS:
			codec.FmtpLine += ";sprop-sps=" + base64.StdEncoding.EncodeToString(payload[4:size])
		case h265.NALUTypePPS:
			codec.FmtpLine += ";sprop-pps=" + base64.StdEncoding.EncodeToString(payload[4:size])
		}
		if size < len(payload) {
			payload = payload[size:]
		} else {
			break
		}
	}
	m.AddTrack(codec)
	return m.getInit()
}

func (m *Muxer) getInit() []byte {

	// tracerr.PrintSourceColor(readNonExistent())

	mv := iso.NewMovie(1024)
	mv.WriteFileType()

	mv.StartAtom(iso.Moov)
	mv.WriteMovieHeader()

	for i, codec := range m.codecs {
		switch codec.Name {
		case core.CodecH265:
			vps, sps, pps := h265.GetParameterSet(codec.FmtpLine)
			// some dummy SPS and PPS not a problem
			if len(vps) == 0 {
				vps = []byte{0x40, 0x01, 0x0c, 0x01, 0xff, 0xff, 0x01, 0x40, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x99, 0xac, 0x09}
			}
			if len(sps) == 0 {
				sps = []byte{0x42, 0x01, 0x01, 0x01, 0x40, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x99, 0xa0, 0x01, 0x40, 0x20, 0x05, 0xa1, 0xfe, 0x5a, 0xee, 0x46, 0xc1, 0xae, 0x55, 0x04}
			}
			if len(pps) == 0 {
				pps = []byte{0x44, 0x01, 0xc0, 0x73, 0xc0, 0x4c, 0x90}
			}

			var width, height uint16
			if s := h265.DecodeSPS(sps); s != nil {
				width = s.Width()
				height = s.Height()
			} else {
				width = 1920
				height = 1080
			}

			mv.WriteVideoTrack(
				uint32(i+1), codec.Name, codec.ClockRate, width, height, h265.EncodeConfig(vps, sps, pps),
			)
		}
	}

	mv.StartAtom(iso.MoovMvex)
	for i := range m.codecs {
		mv.WriteTrackExtend(uint32(i + 1))
	}
	mv.EndAtom() // MVEX

	mv.EndAtom() // MOOV

	return mv.Bytes()
}

func (m *Muxer) GetPayload(trackID byte, payload *[]byte, duration uint32) []byte {
	m.index++
	var flags uint32
	if h265.IsKeyframe(*payload) {
		flags = iso.SampleVideoIFrame
	} else {
		flags = iso.SampleVideoNonIFrame
	}
	size := len(*payload)
	mv := iso.NewMovie(1024 + size)
	// Duration is often calculated as time difference from the previous frame.
	// But for the first frame it will mean 0. This may cause temporary visual glitches e.g. green screen.
	// Patch it:
	if duration == 0 {
		duration = 6000
	}
	mv.WriteMovieFragment(m.index, uint32(trackID+1), duration, uint32(size), flags, m.dts[trackID], uint32(0))
	mv.WriteData(*payload)
	m.dts[trackID] += uint64(duration)
	return mv.Bytes()
}
