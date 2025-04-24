package formats

import (
	"io"
	"log"
	"os"

	"github.com/deepch/vdk/codec/aacparser"
	"github.com/deepch/vdk/codec/h264parser"
	"github.com/deepch/vdk/codec/h265parser"
	"github.com/deepch/vdk/codec/opusparser"
	"github.com/deepch/vdk/format/mkv"
)

func MKVDemuxer(r io.Reader) *mkv.Demuxer {
	return mkv.NewDemuxer(r)
}

func testMKV() {
	fp, err := os.Open("input.mkv")
	if err != nil {
		log.Println(err)
		return
	} else {
		defer fp.Close()
	}

	media := MKVDemuxer(fp)
	if streams, err := media.Streams(); err != nil {
		log.Println(err)
		return
	} else {
		for _, stream := range streams {
			log.Println(stream)

			switch stream := stream.(type) {
			case h264parser.CodecData:
				_ = stream.SPS()
			case h265parser.CodecData:
				_ = stream.VPS()
			case aacparser.CodecData:
			case opusparser.CodecData:
				_ = stream.SampleRate()
			}
		}
	}

	for {
		pkt, err := media.ReadPacket()
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
