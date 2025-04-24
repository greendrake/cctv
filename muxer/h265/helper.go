package h265

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"github.com/greendrake/cctv/muxer/bits"
	"strings"
)

const (
	NALUTypePFrame    = 1
	NALUTypeIFrame    = 19
	NALUTypeIFrame2   = 20
	NALUTypeIFrame3   = 21
	NALUTypeVPS       = 32
	NALUTypeSPS       = 33
	NALUTypePPS       = 34
	NALUTypePrefixSEI = 39
	NALUTypeSuffixSEI = 40
	NALUTypeFU        = 49
)

func Between(s, sub1, sub2 string) string {
	i := strings.Index(s, sub1)
	if i < 0 {
		return ""
	}
	s = s[i+len(sub1):]

	if i = strings.Index(s, sub2); i >= 0 {
		return s[:i]
	}

	return s
}

func NALUType(b []byte) byte {
	return (b[4] >> 1) & 0x3F
}

func IsKeyframe(b []byte) bool {
	for {
		switch NALUType(b) {
		case NALUTypePFrame:
			return false
		case NALUTypeIFrame, NALUTypeIFrame2, NALUTypeIFrame3:
			return true
		}

		size := int(binary.BigEndian.Uint32(b)) + 4
		if size < len(b) {
			b = b[size:]
			continue
		} else {
			return false
		}
	}
}

func GetParameterSet(fmtp string) (vps, sps, pps []byte) {
	if fmtp == "" {
		return
	}

	s := Between(fmtp, "sprop-vps=", ";")
	vps, _ = base64.StdEncoding.DecodeString(s)

	s = Between(fmtp, "sprop-sps=", ";")
	sps, _ = base64.StdEncoding.DecodeString(s)

	s = Between(fmtp, "sprop-pps=", ";")
	pps, _ = base64.StdEncoding.DecodeString(s)

	return
}

func EncodeToAVCC(annexb []byte) (avc []byte) {
	var start int

	avc = make([]byte, 0, len(annexb)+4) // init memory with little overhead

	for i := 0; ; i++ {
		var offset int

		if i+3 < len(annexb) {
			// search next separator
			if annexb[i] == 0 && annexb[i+1] == 0 {
				if annexb[i+2] == 1 {
					offset = 3 // 00 00 01
				} else if annexb[i+2] == 0 && annexb[i+3] == 1 {
					offset = 4 // 00 00 00 01
				} else {
					continue
				}
			} else {
				continue
			}
		} else {
			i = len(annexb) // move i to data end
		}

		if start != 0 {
			size := uint32(i - start)
			avc = binary.BigEndian.AppendUint32(avc, size)
			avc = append(avc, annexb[start:i]...)
		}

		// sometimes FFmpeg put separator at the end
		if i += offset; i == len(annexb) {
			break
		}

		if isAUD(annexb[i]) {
			start = 0 // skip this NALU
		} else {
			start = i // save this position
		}
	}

	return
}

func isAUD(b byte) bool {
	const h264 = 9
	const h265 = 35 << 1
	return b&0b0001_1111 == h264 || b&0b0111_1110 == h265
}

func EncodeConfig(vps, sps, pps []byte) []byte {
	vpsSize := uint16(len(vps))
	spsSize := uint16(len(sps))
	ppsSize := uint16(len(pps))

	buf := make([]byte, 23+5+vpsSize+5+spsSize+5+ppsSize)

	buf[0] = 1
	copy(buf[1:], sps[3:6]) // profile
	buf[21] = 3             // ?
	buf[22] = 3             // ?

	b := buf[23:]
	_ = b[5]
	b[0] = (vps[0] >> 1) & 0x3F
	binary.BigEndian.PutUint16(b[1:], 1) // VPS count
	binary.BigEndian.PutUint16(b[3:], vpsSize)
	copy(b[5:], vps)

	b = buf[23+5+vpsSize:]
	_ = b[5]
	b[0] = (sps[0] >> 1) & 0x3F
	binary.BigEndian.PutUint16(b[1:], 1) // SPS count
	binary.BigEndian.PutUint16(b[3:], spsSize)
	copy(b[5:], sps)

	b = buf[23+5+vpsSize+5+spsSize:]
	_ = b[5]
	b[0] = (pps[0] >> 1) & 0x3F
	binary.BigEndian.PutUint16(b[1:], 1) // PPS count
	binary.BigEndian.PutUint16(b[3:], ppsSize)
	copy(b[5:], pps)

	return buf
}

// http://www.itu.int/rec/T-REC-H.265

//goland:noinspection GoSnakeCaseUsage
type SPS struct {
	sps_video_parameter_set_id   uint8
	sps_max_sub_layers_minus1    uint8
	sps_temporal_id_nesting_flag byte

	general_profile_space               uint8
	general_tier_flag                   byte
	general_profile_idc                 uint8
	general_profile_compatibility_flags uint32

	general_level_idc              uint8
	sub_layer_profile_present_flag []byte
	sub_layer_level_present_flag   []byte

	sps_seq_parameter_set_id   uint32
	chroma_format_idc          uint32
	separate_colour_plane_flag byte

	pic_width_in_luma_samples  uint32
	pic_height_in_luma_samples uint32
}

func (s *SPS) Width() uint16 {
	return uint16(s.pic_width_in_luma_samples)
}

func (s *SPS) Height() uint16 {
	return uint16(s.pic_height_in_luma_samples)
}

func DecodeSPS(nalu []byte) *SPS {
	rbsp := bytes.ReplaceAll(nalu[2:], []byte{0, 0, 3}, []byte{0, 0})

	r := bits.NewReader(rbsp)
	s := &SPS{}

	s.sps_video_parameter_set_id = r.ReadBits8(4)
	s.sps_max_sub_layers_minus1 = r.ReadBits8(3)
	s.sps_temporal_id_nesting_flag = r.ReadBit()

	if !s.profile_tier_level(r) {
		return nil
	}

	s.sps_seq_parameter_set_id = r.ReadUEGolomb()
	s.chroma_format_idc = r.ReadUEGolomb()
	if s.chroma_format_idc == 3 {
		s.separate_colour_plane_flag = r.ReadBit()
	}

	s.pic_width_in_luma_samples = r.ReadUEGolomb()
	s.pic_height_in_luma_samples = r.ReadUEGolomb()

	//...

	if r.EOF {
		return nil
	}

	return s
}

// profile_tier_level supports ONLY general_profile_idc == 1
// over variants very complicated...
//
//goland:noinspection GoSnakeCaseUsage
func (s *SPS) profile_tier_level(r *bits.Reader) bool {
	s.general_profile_space = r.ReadBits8(2)
	s.general_tier_flag = r.ReadBit()
	s.general_profile_idc = r.ReadBits8(5)

	s.general_profile_compatibility_flags = r.ReadBits(32)
	_ = r.ReadBits64(48) // other flags

	if s.general_profile_idc != 1 {
		return false
	}

	s.general_level_idc = r.ReadBits8(8)

	s.sub_layer_profile_present_flag = make([]byte, s.sps_max_sub_layers_minus1)
	s.sub_layer_level_present_flag = make([]byte, s.sps_max_sub_layers_minus1)

	for i := byte(0); i < s.sps_max_sub_layers_minus1; i++ {
		s.sub_layer_profile_present_flag[i] = r.ReadBit()
		s.sub_layer_level_present_flag[i] = r.ReadBit()
	}

	if s.sps_max_sub_layers_minus1 > 0 {
		for i := s.sps_max_sub_layers_minus1; i < 8; i++ {
			_ = r.ReadBits8(2) // reserved_zero_2bits
		}
	}

	for i := byte(0); i < s.sps_max_sub_layers_minus1; i++ {
		if s.sub_layer_profile_present_flag[i] != 0 {
			_ = r.ReadBits8(2)                      // sub_layer_profile_space
			_ = r.ReadBit()                         // sub_layer_tier_flag
			sub_layer_profile_idc := r.ReadBits8(5) // sub_layer_profile_idc

			_ = r.ReadBits(32)   // sub_layer_profile_compatibility_flag
			_ = r.ReadBits64(48) // other flags

			if sub_layer_profile_idc != 1 {
				return false
			}
		}

		if s.sub_layer_level_present_flag[i] != 0 {
			_ = r.ReadBits8(8)
		}
	}

	return true
}
