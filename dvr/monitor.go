package dvrip

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/greendrake/cctv/dvr/frame"
	"github.com/greendrake/cctv/dvr/packet"
	gframe "github.com/greendrake/cctv/frame"
	"github.com/greendrake/eventbus"
	"github.com/greendrake/fractions"
	"io"
	"math"
	"time"
)

type Monitor struct {
	eventbus.EventBus
	client         *Client
	sType          string
	lastIFrameMeta *frame.Meta
	pts            *PTS
	claimDone      bool
	pps            []byte
	sps            []byte
}

func NewMonitor(ctx context.Context, address string, sType string, args ...string) (*Monitor, error) {
	client, err := NewClient(ctx, address, args...)
	if err != nil {
		return nil, err
	}
	sMap := map[string]string{
		"0": "Main",
		"1": "Extra",
	}
	return &Monitor{
		client:    client,
		sType:     sMap[sType],
		claimDone: false,
	}, nil
}

func (me *Monitor) ShutDown() {
	if me.client != nil {
		me.client.Disconnect()
		me.client = nil
	}
}

func (me *Monitor) GetFrame() (*gframe.Frame, error) {
	if !me.claimDone {
		err := me.client.Monitor(me.sType)
		if err == nil {
			me.claimDone = true
		} else {
			return nil, err
		}
	}
	raw, err := me.getRawFrame()
	if err != nil {
		return nil, err
	}
	if raw.Type == frame.T_VideoI {
		me.lastIFrameMeta = raw.GetMeta()
		if me.pts == nil || me.pts.FPS != me.lastIFrameMeta.FPS {
			me.pts, err = NewPTS(me.lastIFrameMeta.FPS)
			if err != nil {
				return nil, err
			}
		}
	} else if raw.Type == frame.T_VideoP {
		if me.pts == nil {
			return nil, fmt.Errorf("P frame came first before I frame")
		}
		meta := *me.lastIFrameMeta
		meta.Type = raw.Type
		raw.Meta = &meta
	}
	meta := raw.GetMeta()
	F := &gframe.Frame{
		Data: raw.Data,
	}
	if raw.Type == frame.T_VideoI || raw.Type == frame.T_VideoP {
		F.Duration = time.Duration(me.pts.Next()) * time.Millisecond
		F.IsVideo = true
		F.IsVideoKeyFrame = raw.Type == frame.T_VideoI
		F.IsHEVC = meta.MediaType == "H265"
	} else if raw.Type == frame.T_Audio {
		if me.pts == nil {
			return nil, fmt.Errorf("Audio frame came first before I frame")
		}
		F.Duration = time.Duration(1000*len(*raw.Data)/int(meta.SampleRate)) * time.Millisecond
		F.IsAudio = true
	}
	return F, nil
}

func (me *Monitor) getRawFrame() (*frame.RawFrame, error) {
	me.client.MaybePingKeepAlive()
	message, err := me.client.GetMessage()
	if err != nil {
		return nil, err
	}
	if message.Code == packet.KEEPALIVE_RSP {
		// Ignore and read the next one
		return me.getRawFrame()
	}
	if message.Code != packet.MONITOR_DATA { // read until get a media message
		return nil, fmt.Errorf("Expected monitor data but received code %d", message.Code)
	}
	reader := bytes.NewReader(message.Data)
	var headerTop frame.HeaderCommon
	err = binary.Read(reader, binary.LittleEndian, &headerTop) // here it doesn't actually matter whether Big or Little Endian as all the fields are 1-byte long
	if err != nil {
		return nil, err
	}
	if headerTop.B1 != 0x00 || headerTop.B2 != 0x00 || headerTop.B3 != 0x01 {
		// fmt.Println("Wrong first 3 bytes of a what is expected to be a frame")
		// Ignore and read the next one
		return me.getRawFrame()
	}
	header := frame.TMap[headerTop.Type]()
	err = binary.Read(reader, binary.LittleEndian, header)
	if err != nil {
		return nil, err
	}
	factualDataLength := uint32(len(message.Data) - int(header.GetOwnLength()))
	if header.GetLength() != factualDataLength {
		diff := factualDataLength - header.GetLength()
		if diff != 168 { // 168 appears to be frequent and benign
			return nil, fmt.Errorf("Frame data length unexpected. Got: %d. Expected as per the media header: %d; Diff: %d\n", factualDataLength, header.GetLength(), diff)
		}
	}
	data := make([]byte, factualDataLength)
	n, err := io.ReadFull(reader, data)
	if err != nil {
		return nil, err
	}
	if uint32(n) != factualDataLength {
		return nil, fmt.Errorf("Could not read the correct amount of bytes of frame data. Expected: %d. Got: %d\n", factualDataLength, n)
	}
	return &frame.RawFrame{
		Type:   headerTop.Type,
		Header: header,
		Data:   &data,
	}, nil
}

type PTS struct {
	FPS          uint8
	numerator    uint8
	denominator  uint8
	currentIndex uint8
	loSpf        uint8
	hiSpf        uint8
	isRound      bool
}

func isRound(number float64) bool {
	// Compare the original number with its rounded or truncated value
	return number == math.Round(number) || number == math.Trunc(number)
}

func NewPTS(fps uint8) (*PTS, error) {
	pts := &PTS{
		FPS: fps,
	}
	spf := float64(1000 / float64(fps))
	pts.isRound = isRound(spf)
	if pts.isRound {
		pts.loSpf = uint8(spf)
	} else {
		pts.loSpf = uint8(math.Floor(spf))
		pts.hiSpf = pts.loSpf + 1
		frac, err := fractions.FloatToFrac(spf - float64(pts.loSpf))
		if err != nil {
			panic(fmt.Sprintf("Could not create fraction for FPS %d: %s", fps, err))
		}
		pts.numerator = uint8(fractions.GetNumerator(frac))
		pts.denominator = uint8(fractions.GetDenominator(frac))
		pts.currentIndex = 1
	}
	return pts, nil
}

func (pts *PTS) Next() uint8 {
	if pts.isRound {
		return pts.loSpf
	}
	var next uint8
	if pts.currentIndex <= pts.numerator {
		next = pts.hiSpf
	} else {
		next = pts.loSpf
	}
	if pts.currentIndex == pts.denominator {
		// rewind
		pts.currentIndex = 1
	} else {
		pts.currentIndex++
	}
	return next
}
