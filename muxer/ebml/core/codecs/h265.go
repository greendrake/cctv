package codecs

import (
	"bytes"
	"fmt"
	"io"

	"gitee.com/general252/gomedia/go-rtsp/sdp"
	"github.com/bluenviron/mediacommon/pkg/codecs/h265"
	"github.com/deepch/vdk/codec/h265parser"
)

func EmitNALUH265Data(data []byte, withStartCode NALUFormatType, emit func(t h265.NALUType, data []byte)) error {
	r := bytes.NewReader(data)
	_, tye := GetNALUFormatType(data)
	if tye == NALUFormatNo {
		return fmt.Errorf("unknown type")
	}

	return EmitNALUH265Reader(r, tye, withStartCode, emit)
}

func EmitNALUH265Reader(r io.Reader, typ NALUFormatType, withStartCode NALUFormatType, emit func(t h265.NALUType, data []byte)) error {
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
		t := h265.NALUType((firstByte & 0x7E) >> 1)

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

func IsH265KeyFrame(t h265.NALUType) bool {
	// P: NALUType_TRAIL_R
	if t == h265.NALUType_IDR_W_RADL || t == h265.NALUType_IDR_N_LP || t == h265.NALUType_CRA_NUT {
		return true
	}
	return false
}

func H265NALUType(firstByte byte) h265.NALUType {
	t := h265.NALUType((firstByte & 0x7E) >> 1)
	return t
}

type H265Param struct {
	vps []byte
	sps []byte
	pps []byte
}

func NewH265Param(vps []byte, sps []byte, pps []byte) *H265Param {
	return &H265Param{vps: vps, sps: sps, pps: pps}
}

func (c *H265Param) GetFmtpString() string {
	var opts = []sdp.H265FmtpPramOption{
		sdp.WithH265VPS(c.vps),
		sdp.WithH265SPS(c.sps),
		sdp.WithH265PPS(c.pps),
	}

	param := sdp.NewH265FmtpParam(opts...)
	return param.Save()
}

func (c *H265Param) Load(fmtp string) {
	var param = sdp.NewH265FmtpParam()
	param.Load(fmtp)

	c.vps, c.sps, c.pps = param.GetVpsSpsPps()
}

func (c *H265Param) GetVpsSpsPps() ([]byte, []byte, []byte) {
	return c.vps, c.sps, c.pps
}

func (c *H265Param) ParseSPS() (h265parser.SPSInfo, error) {
	return h265parser.ParseSPS(c.sps)
}

func (c *H265Param) GetExtraData() ([]byte, error) {
	codecData, err := h265parser.NewCodecDataFromVPSAndSPSAndPPS(c.vps, c.sps, c.pps)
	if err != nil {
		return nil, err
	}

	return codecData.AVCDecoderConfRecordBytes(), nil
}

func (c *H265Param) GetCodecData() (h265parser.CodecData, error) {
	return h265parser.NewCodecDataFromVPSAndSPSAndPPS(c.vps, c.sps, c.pps)
}
