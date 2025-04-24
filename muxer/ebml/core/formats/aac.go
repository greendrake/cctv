package formats

import (
	"fmt"
	"io"
	"log"
	"os"

	"gitee.com/general252/gomedia/go-codec"
	"github.com/bluenviron/mediacommon/pkg/codecs/mpeg4audio"
	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/format/aac"
)

var sampleRates = []int{
	96000,
	88200,
	64000,
	48000,
	44100,
	32000,
	24000,
	22050,
	16000,
	12000,
	11025,
	8000,
	7350,
}

// EmitADTSReader decodes an ADTS stream into ADTS packets.
func EmitADTSReader(r io.Reader, emit func(pkt *mpeg4audio.ADTSPacket, data []byte, header []byte)) error {
	// refs: https://wiki.multimedia.cx/index.php/ADTS

	for {
		var (
			bl  int
			pos int
			buf = make([]byte, 7)
			err error
		)
		if bl, err = r.Read(buf); err != nil {
			return err
		}

		if (bl - pos) < 7 {
			return fmt.Errorf("invalid length")
		}

		syncWord := (uint16(buf[pos]) << 4) | (uint16(buf[pos+1]) >> 4)
		if syncWord != 0xfff {
			return fmt.Errorf("invalid syncword")
		}

		protectionAbsent := buf[pos+1] & 0x01
		if protectionAbsent != 1 {
			return fmt.Errorf("CRC is not supported")
		}

		pkt := &mpeg4audio.ADTSPacket{}

		pkt.Type = mpeg4audio.ObjectType((buf[pos+2] >> 6) + 1)
		switch pkt.Type {
		case mpeg4audio.ObjectTypeAACLC:
		default:
			return fmt.Errorf("unsupported audio type: %d", pkt.Type)
		}

		sampleRateIndex := (buf[pos+2] >> 2) & 0x0F
		switch {
		case sampleRateIndex <= 12:
			pkt.SampleRate = sampleRates[sampleRateIndex]

		default:
			return fmt.Errorf("invalid sample rate index: %d", sampleRateIndex)
		}

		channelConfig := ((buf[pos+2] & 0x01) << 2) | ((buf[pos+3] >> 6) & 0x03)
		switch {
		case channelConfig >= 1 && channelConfig <= 6:
			pkt.ChannelCount = int(channelConfig)

		case channelConfig == 7:
			pkt.ChannelCount = 8

		default:
			return fmt.Errorf("invalid channel configuration: %d", channelConfig)
		}

		frameLen := int(((uint16(buf[pos+3])&0x03)<<11)|
			(uint16(buf[pos+4])<<3)|
			((uint16(buf[pos+5])>>5)&0x07)) - 7

		if frameLen <= 0 {
			return fmt.Errorf("invalid FrameLen")
		}

		if frameLen > mpeg4audio.MaxAccessUnitSize {
			return fmt.Errorf("access unit size (%d) is too big, maximum is %d", frameLen, mpeg4audio.MaxAccessUnitSize)
		}

		frameCount := buf[pos+6] & 0x03
		if frameCount != 0 {
			return fmt.Errorf("frame count greater than 1 is not supported")
		}

		pkt.AU = make([]byte, frameLen)
		if n, err := r.Read(pkt.AU); err != nil {
			if err == io.EOF {
				break
			}
			return err
		} else if n < frameLen {
			return fmt.Errorf("invalid frame length")
		}

		emit(pkt, pkt.AU, buf)
	}

	return nil
}

func AACDemuxer(r io.Reader) *aac.Demuxer {
	return aac.NewDemuxer(r)
}

func testAAC() {
	fp, err := os.Open("input.mkv")
	if err != nil {
		log.Println(err)
		return
	} else {
		defer fp.Close()
	}

	demuxer := AACDemuxer(fp)
	{
		streams, err := demuxer.Streams()
		if err != nil {
			log.Println(err)
			return
		}

		for _, stream := range streams {
			if stream.Type() == av.AAC {
				switch stream := stream.(type) {
				case av.AudioCodecData:
					_ = stream
				}
			}
		}
	}

	for {
		pkt, err := demuxer.ReadPacket()
		if err != nil {
			log.Println(err)
			break
		}

		_ = pkt.IsKeyFrame
		_ = pkt.Idx
		_ = pkt.Time
		_ = pkt.Data
	}
}

func ConvertADTSToASC(frame []byte) ([]byte, error) {
	object, err := codec.ConvertADTSToASC(frame)
	if err != nil {
		return nil, err
	}

	return object.Encode(), nil
}

func ConvertASCToADTS(asc []byte, aacbytes int) ([]byte, error) {
	object, err := codec.ConvertASCToADTS(asc, aacbytes)
	if err != nil {
		return nil, err
	}

	return object.Encode(), nil
}
