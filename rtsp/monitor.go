package rtsp

import (
	// "log"
	"context"
	"errors"
	"time"
	// "strings"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/pkg/codecs/h265"
	"github.com/greendrake/cctv/frame"
	"github.com/pion/rtp"
)

type Monitor struct {
	client       *gortsplib.Client
	Ctx          context.Context
	frameChannel chan *frame.Frame
	vps          []byte
	sps          []byte
	pps          []byte
}

func NewMonitor(ctx context.Context, address string) (*Monitor, error) {
	frameChannel := make(chan *frame.Frame)
	c := &gortsplib.Client{
		OnPacketLost: func(err error) {
			// ignore
		},
		OnDecodeError: func(err error) {
			// ignore
		},
	}

	// parse URL
	// log.Println(address)
	u, err := base.ParseURL(address)
	if err != nil {
		return nil, err
	}

	// connect to the server
	err = c.Start(u.Scheme, u.Host)
	if err != nil {
		return nil, err
	}

	// find available medias
	desc, _, err := c.Describe(u)
	if err != nil {
		return nil, err
	}

	// find the H264 media and format
	var forma *format.H265
	medi := desc.FindFormat(&forma)
	if medi == nil {
		panic("media not found")
	}

	// setup RTP -> H264 decoder
	rtpDec, err := forma.CreateDecoder()
	if err != nil {
		return nil, err
	}

	// setup a single media
	_, err = c.Setup(desc.BaseURL, medi, 0, 0)
	if err != nil {
		panic(err)
	}

	monitor := &Monitor{
		client:       c,
		Ctx:          ctx,
		frameChannel: frameChannel,
	}

	var prevPTS time.Duration

	// called when a RTP packet arrives
	c.OnPacketRTP(medi, forma, func(pkt *rtp.Packet) {
		// decode timestamp
		pts, ok := c.PacketPTS(medi, pkt)
		if !ok {
			return
		}

		// extract access unit from RTP packets
		au, err := rtpDec.Decode(pkt)
		if err != nil {
			return
		}

		f, err := monitor.buildH265Frame(au, pts-prevPTS)
		prevPTS = pts
		if err != nil {
			panic(err)
		}

		frameChannel <- f
	})

	_, err = c.Play(nil)
	if err != nil {
		return nil, err
	}

	return monitor, err
}

func (me *Monitor) ShutDown() {
	if me.client != nil {
		me.client.Close()
		me.client = nil
	}
}

func (me *Monitor) GetFrame() (*frame.Frame, error) {
	timeout := time.After(5 * time.Second)
	select {
	case <-timeout:
		return nil, errors.New("Timeout")
	case <-me.Ctx.Done():
		return nil, errors.New("Context done")
	case f := <-me.frameChannel:
		return f, nil
	}
}

// ported from gortsplib
func (m *Monitor) buildH264Frame(au [][]byte, pts time.Duration) (*frame.Frame, error) {
	var filteredAU [][]byte

	nonIDRPresent := false
	idrPresent := false

	for _, nalu := range au {
		typ := h264.NALUType(nalu[0] & 0x1F)
		switch typ {
		case h264.NALUTypeSPS:
			m.sps = nalu
			continue

		case h264.NALUTypePPS:
			m.pps = nalu
			continue

		case h264.NALUTypeAccessUnitDelimiter:
			continue

		case h264.NALUTypeIDR:
			idrPresent = true

		case h264.NALUTypeNonIDR:
			nonIDRPresent = true
		}

		filteredAU = append(filteredAU, nalu)
	}

	au = filteredAU

	if au == nil || (!nonIDRPresent && !idrPresent) {
		return nil, nil
	}

	// add SPS and PPS before access unit that contains an IDR
	if idrPresent {
		au = append([][]byte{m.sps, m.pps}, au...)
	}

	enc, err := h264.AnnexBMarshal(au)
	if err != nil {
		return nil, err
	}
	return &frame.Frame{
		IsVideo:         true,
		IsVideoKeyFrame: idrPresent,
		Duration:        pts,
		Data:            &enc,
	}, nil
}

func (m *Monitor) buildH265Frame(au [][]byte, pts time.Duration) (*frame.Frame, error) {
	var filteredAU [][]byte

	isRandomAccess := false

	for _, nalu := range au {
		typ := h265.NALUType((nalu[0] >> 1) & 0b111111)
		switch typ {
		case h265.NALUType_VPS_NUT:
			m.vps = nalu
			continue

		case h265.NALUType_SPS_NUT:
			m.sps = nalu
			continue

		case h265.NALUType_PPS_NUT:
			m.pps = nalu
			continue

		case h265.NALUType_AUD_NUT:
			continue

		case h265.NALUType_IDR_W_RADL, h265.NALUType_IDR_N_LP, h265.NALUType_CRA_NUT:
			isRandomAccess = true
		}

		filteredAU = append(filteredAU, nalu)
	}

	au = filteredAU

	if au == nil {
		return nil, nil
	}

	// add VPS, SPS and PPS before random access access unit
	if isRandomAccess {
		au = append([][]byte{m.vps, m.sps, m.pps}, au...)
	}

	enc, err := h264.AnnexBMarshal(au)
	if err != nil {
		return nil, err
	}
	return &frame.Frame{
		IsHEVC:          true,
		IsVideo:         true,
		IsVideoKeyFrame: isRandomAccess,
		Duration:        pts,
		Data:            &enc,
	}, nil
}
