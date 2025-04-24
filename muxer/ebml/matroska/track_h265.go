package matroska

import (
	"bytes"
	"log"
	"time"

	"github.com/bluenviron/mediacommon/pkg/codecs/h265"
	"github.com/greendrake/cctv/muxer/ebml/core"
	"github.com/greendrake/cctv/muxer/ebml/core/codecs"
	"github.com/greendrake/cctv/muxer/ebml/webm"
)

type trackH265 struct {
	UnimplementedTrack

	vps      bytes.Buffer
	sps      bytes.Buffer
	pps      bytes.Buffer
	sei      bytes.Buffer
	setPixel bool

	idrWRadl bytes.Buffer
}

func NewTrackH265(opts ...Option[*trackH265]) Track {
	const CodecPrivateSize = 255
	t := &trackH265{
		UnimplementedTrack: UnimplementedTrack{
			track: webm.TrackEntry{
				Name:            "Video(HEVC)",
				TrackNumber:     1,
				TrackUID:        getTrackUID(),
				CodecID:         core.VideoCodecMPEGHISOHEVC,
				TrackType:       1,
				DefaultDuration: 40000000,
				Video: &webm.Video{
					PixelWidth:  1,
					PixelHeight: 1,
					Void:        make([]byte, 8),
				},
				CodecPrivate: make([]byte, 2),
				Void:         make([]byte, CodecPrivateSize-2),
			},
		},
	}

	for _, opt := range opts {
		opt.Apply(t)
	}

	return t
}

// Write format: AnnexB/AVCC
func (tis *trackH265) Write(timestamp time.Duration, b []byte, _ ...bool) (int, error) {
	if len(b) < 5 {
		return 0, ErrH264PacketSize
	}

	var (
		err   error
		n     int
		count int

		buffer bytes.Buffer
	)

	err = codecs.EmitNALUH265Data(b, codecs.NALUFormatAVCC, func(t h265.NALUType, data []byte) {
		if err != nil {
			return
		}

		switch t {
		case h265.NALUType_VPS_NUT:
			tis.updateVPS(data)
			return
		case h265.NALUType_SPS_NUT:
			tis.updateSPS(data)
			return
		case h265.NALUType_PPS_NUT:
			tis.updatePPS(data)
			return
		case h265.NALUType_PREFIX_SEI_NUT:
			tis.updateSEI(data)
			return
		default:
		}

		switch t {
		case h265.NALUType_IDR_W_RADL:
			// 缓存IDR帧
			tis.idrWRadl.Write(data)
			return
		default:
			if tis.idrWRadl.Len() > 0 {
				tmpData := tis.formatIDR(tis.idrWRadl.Bytes())
				if count, err = tis.UnimplementedTrack.Write(timestamp, tmpData, true); err == nil {
					n += count
				}

				tis.idrWRadl.Reset()
			}
		}

		buffer.Write(data)
	})

	if buffer.Len() > 0 {
		if count, err = tis.UnimplementedTrack.Write(timestamp, buffer.Bytes()); err == nil {
			n += count
		}
	}

	if err != nil {
		return 0, err
	}

	return n, nil
}

func (tis *trackH265) formatIDR(data []byte) []byte {
	var buff bytes.Buffer
	if tis.vps.Len() > 0 {
		buff.Write(tis.vps.Bytes())
	}
	if tis.sps.Len() > 0 {
		buff.Write(tis.sps.Bytes())
	}
	if tis.pps.Len() > 0 {
		buff.Write(tis.pps.Bytes())
	}
	if tis.sei.Len() > 0 {
		buff.Write(tis.sei.Bytes())
	}

	if tis.vps.Len() > 4 && tis.sps.Len() > 4 && tis.pps.Len() > 4 {
		h := codecs.NewH265Param(tis.vps.Bytes()[4:], tis.sps.Bytes()[4:], tis.pps.Bytes()[4:])
		if data, err := h.GetExtraData(); err != nil {
			log.Println(err)
		} else {
			if err = tis.track.SetCodecPrivate(data); err != nil {
				log.Println(err)
			}
		}
	}

	buff.Write(data)

	return buff.Bytes()
}

func (tis *trackH265) updateSPS(data []byte) {
	if len(data) > 0 {
		tis.sps.Reset()
		tis.sps.Write(data)
	}

	if !tis.setPixel && tis.sps.Len() > 4 {
		_ = codecs.EmitNALUH265Data(tis.sps.Bytes(), codecs.NALUFormatNo, func(t h265.NALUType, data []byte) {
			if t != h265.NALUType_SPS_NUT {
				return
			}

			var object h265.SPS
			if err := object.Unmarshal(data); err == nil {
				tis.track.Video.Set(uint64(object.Width()), uint64(object.Height()))

				tis.setPixel = true
			}
		})

	}
}

func (tis *trackH265) updateVPS(data []byte) {
	if len(data) > 0 {
		tis.vps.Reset()
		tis.vps.Write(data)
	}
}

func (tis *trackH265) updatePPS(data []byte) {
	if len(data) > 0 {
		tis.pps.Reset()
		tis.pps.Write(data)
	}
}

func (tis *trackH265) updateSEI(data []byte) {
	if len(data) > 0 {
		tis.sei.Reset()
		tis.sei.Write(data)
	}
}

// WithH265SPSPPS vps/sps/pps 00 00 00 01 ...
func WithH265SPSPPS(vps, sps, pps []byte) Option[*trackH265] {
	return NewFuncOption(func(o *trackH265) {
		o.updateVPS(vps)
		o.updateSPS(sps)
		o.updatePPS(pps)
	})
}
