package matroska

import (
	"bytes"
	"log"
	"time"

	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"github.com/greendrake/cctv/muxer/ebml/core"
	"github.com/greendrake/cctv/muxer/ebml/core/codecs"
	"github.com/greendrake/cctv/muxer/ebml/webm"
)

// ffmpeg -i 3840x2160.mp4 -c:v libx264 -x264-params "slice-max-size=5000" -an 3840x2160.h264
// ffmpeg -i 3840x2160.mp4 -c:v libx264 -x264-params "slices=4" -an 3840x2160.h264

type trackH264 struct {
	UnimplementedTrack

	sps      bytes.Buffer
	pps      bytes.Buffer
	sei      bytes.Buffer
	setPixel bool

	idrWRadl bytes.Buffer
}

func NewTrackH264(opts ...Option[*trackH264]) Track {
	const CodecPrivateSize = 255
	t := &trackH264{
		UnimplementedTrack: UnimplementedTrack{
			track: webm.TrackEntry{
				Name:        "Video(H264)",
				TrackNumber: 1,
				TrackUID:    getTrackUID(),
				CodecID:     core.VideoCodecMPEG4ISOAVC,
				TrackType:   core.TrackTypeVideo,
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
func (tis *trackH264) Write(timestamp time.Duration, b []byte, _ ...bool) (int, error) {
	if len(b) < 5 {
		return 0, ErrH264PacketSize
	}

	var (
		err   error
		n     int
		count int

		buffer bytes.Buffer
	)

	err = codecs.EmitNALUH264Data(b, codecs.NALUFormatAVCC, func(t h264.NALUType, data []byte) {
		if err != nil {
			return
		}

		switch t {
		case h264.NALUTypeSPS:
			tis.updateSPS(data)
			return
		case h264.NALUTypePPS:
			tis.updatePPS(data)
			return
		case h264.NALUTypeSEI:
			tis.updateSEI(data)
			return
		default:

		}

		switch t {
		case h264.NALUTypeIDR:
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

func (tis *trackH264) formatIDR(data []byte) []byte {
	var buff bytes.Buffer
	if tis.sps.Len() > 0 {
		buff.Write(tis.sps.Bytes())
	}
	if tis.pps.Len() > 0 {
		buff.Write(tis.pps.Bytes())
	}
	if tis.sei.Len() > 0 {
		buff.Write(tis.sei.Bytes())
	}

	if tis.sps.Len() > 4 && tis.pps.Len() > 4 {
		h := codecs.NewH264Param(tis.sps.Bytes()[4:], tis.pps.Bytes()[4:], 1)
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

func (tis *trackH264) updateSPS(data []byte) {
	if len(data) > 0 {
		tis.sps.Reset()
		tis.sps.Write(data)
	}

	if !tis.setPixel && tis.sps.Len() > 0 {
		_ = codecs.EmitNALUH264Data(tis.sps.Bytes(), codecs.NALUFormatNo, func(t h264.NALUType, data []byte) {
			if t != h264.NALUTypeSPS {
				return
			}

			var object h264.SPS
			if err := object.Unmarshal(data); err == nil {
				tis.track.Video.Set(uint64(object.Width()), uint64(object.Height()))

				tis.setPixel = true
			}
		})
	}
}

func (tis *trackH264) updatePPS(data []byte) {
	if len(data) > 0 {
		tis.pps.Reset()
		tis.pps.Write(data)
	}
}

func (tis *trackH264) updateSEI(data []byte) {
	if len(data) > 0 {
		tis.sei.Reset()
		tis.sei.Write(data)
	}
}

// WithH264SPSPPS sps/pps 00 00 00 01 ...
func WithH264SPSPPS(sps, pps []byte) Option[*trackH264] {
	return NewFuncOption(func(o *trackH264) {
		o.updateSPS(sps)
		o.updatePPS(pps)
	})
}
