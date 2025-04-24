package codecs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"gitee.com/general252/gomedia/go-rtsp/sdp"
	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"github.com/deepch/vdk/codec/h264parser"
)

func EmitNALUH264Data(data []byte, withStartCode NALUFormatType, emit func(t h264.NALUType, data []byte)) error {
	r := bytes.NewReader(data)
	_, tye := GetNALUFormatType(data)
	if tye == NALUFormatNo {
		return fmt.Errorf("unknown type")
	}

	return EmitNALUH264Reader(r, tye, withStartCode, emit)
}

func EmitNALUH264Reader(r io.Reader, typ NALUFormatType, withStartCode NALUFormatType, emit func(t h264.NALUType, data []byte)) error {
	emitFunc := func(data []byte) {
		if len(data) == 0 {
			return
		}

		firstByte := data[0]
		switch withStartCode {
		case NALUFormatNo:
		case NALUFormatAVCC, NALUFormatAnnexB:
			firstByte = data[4]
		}
		t := h264.NALUType((firstByte & 0x1F) >> 0)

		emit(t, data)
	}

	switch typ {
	case NALUFormatAnnexB:
		EmitNALUReaderAnnexB(r, withStartCode, emitFunc)
		return nil
	case NALUFormatAVCC:
		return EmitNALUReaderAVCC(r, withStartCode, emitFunc)
	default:
		return fmt.Errorf("unknown type")
	}
}

func IsH264KeyFrame(t h264.NALUType) bool {
	return t == h264.NALUTypeIDR
}

func H264NALUType(firstByte byte) h264.NALUType {
	t := h264.NALUType((firstByte & 0x1F) >> 0)
	return t
}

// IsRTPKeyFrame data: rtp.Packet.Payload  github.com\pion\webrtc\v3@v3.1.43\pkg\media\h264writer
func IsRTPKeyFrame(data []byte) bool {
	const (
		typeSTAPA       = 24
		typeSPS         = 7
		naluTypeBitmask = 0x1F
	)

	var word uint32

	payload := bytes.NewReader(data)
	if err := binary.Read(payload, binary.BigEndian, &word); err != nil {
		return false
	}

	naluType := (word >> 24) & naluTypeBitmask
	if naluType == typeSTAPA && word&naluTypeBitmask == typeSPS {
		return true
	} else if naluType == typeSPS {
		return true
	}

	return false
}

type H264Param struct {
	packetMode int    // 1,0
	sps        []byte //
	pps        []byte //
}

func NewH264Param(sps []byte, pps []byte, packetMode ...int) *H264Param {
	c := &H264Param{
		packetMode: 0,
		sps:        sps,
		pps:        pps,
	}

	if len(packetMode) > 0 {
		c.packetMode = packetMode[0]
	}

	return c
}

func (c *H264Param) GetFmtpString() string {
	var opts = []sdp.H264ExtraOption{
		sdp.WithPacketizationMode(c.packetMode),
		sdp.WithProfileLevelId(c.sps[1:4]),
		sdp.WithH264SPS(c.sps),
		sdp.WithH264PPS(c.pps),
	}

	param := sdp.NewH264FmtpParam(opts...)
	return param.Save()
}

// Load a=fmtp:98 profile-level-id=42A01E;packetization-mode=1;sprop-parameter-sets=<parameter sets data>
func (c *H264Param) Load(fmtp string) {
	var param = sdp.NewH264FmtpParam()
	param.Load(fmtp)

	c.sps, c.pps = param.GetSpsPps()
}

func (c *H264Param) GetSpsPps() ([]byte, []byte) {
	return c.sps, c.pps
}

func (c *H264Param) ParseSPS() (h264parser.SPSInfo, error) {
	return h264parser.ParseSPS(c.sps)
}

func (c *H264Param) GetExtraData() ([]byte, error) {
	codecData, err := h264parser.NewCodecDataFromSPSAndPPS(c.sps, c.pps)
	if err != nil {
		return nil, err
	}

	return codecData.AVCDecoderConfRecordBytes(), nil
}

func (c *H264Param) GetCodecData() (h264parser.CodecData, error) {
	return h264parser.NewCodecDataFromSPSAndPPS(c.sps, c.pps)
}
