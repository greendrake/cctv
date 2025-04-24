package formats

import (
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/codec/aacparser"
	"github.com/deepch/vdk/codec/h264parser"
	"github.com/deepch/vdk/codec/h265parser"
	"github.com/deepch/vdk/codec/opusparser"
	"github.com/deepch/vdk/format/mkv"
	"github.com/greendrake/cctv/muxer/ebml/core/codecs"

	mp4gomedia "gitee.com/general252/gomedia/go-mp4"
	mp4go2rtc "github.com/AlexxIT/go2rtc/pkg/mp4"
	mp4ffmp4 "github.com/Eyevinn/mp4ff/mp4"
	mp4vdk "github.com/deepch/vdk/format/mp4"
)

func MP4VDKDemuxer(r io.ReadSeeker) *mp4vdk.Demuxer {
	return mp4vdk.NewDemuxer(r)
}

// MP4VDKMuxer vdk mp4 muxer(H264/H265/AAC)
func MP4VDKMuxer(r io.WriteSeeker) *mp4vdk.Muxer {
	return mp4vdk.NewMuxer(r)
}

func MP4GoMediaDeMuxer(r io.ReadSeeker) *mp4gomedia.MovDemuxer {
	return mp4gomedia.CreateMp4Demuxer(r)
}

type Mp4Muxer struct {
	m   *mp4gomedia.Movmuxer
	mux sync.Mutex
}

func (m *Mp4Muxer) AddAudioTrack(cid mp4gomedia.MP4_CODEC_TYPE, options ...mp4gomedia.TrackOption) uint32 {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.m.AddAudioTrack(cid, options...)
}
func (m *Mp4Muxer) AddVideoTrack(cid mp4gomedia.MP4_CODEC_TYPE, options ...mp4gomedia.TrackOption) uint32 {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.m.AddVideoTrack(cid, options...)
}
func (m *Mp4Muxer) Write(track uint32, data []byte, pts uint64, dts uint64) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.m.Write(track, data, pts, dts)
}
func (m *Mp4Muxer) WriteTrailer() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.m.WriteTrailer()
}
func (m *Mp4Muxer) ReBindWriter(w io.WriteSeeker) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.m.ReBindWriter(w)
}
func (m *Mp4Muxer) OnNewFragment(onFragment mp4gomedia.OnFragment) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.m.OnNewFragment(onFragment)
}
func (m *Mp4Muxer) WriteInitSegment(w io.Writer) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.m.WriteInitSegment(w)
}
func (m *Mp4Muxer) FlushFragment() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.m.FlushFragment()
}

// MP4GoMediaMuxer gomedia mp4(H264/H265/AAC/G711/MP4_CODEC_MP3)
func MP4GoMediaMuxer(w io.WriteSeeker) (*Mp4Muxer, error) {
	m, err := mp4gomedia.CreateMp4Muxer(w, mp4gomedia.WithMp4Flag(mp4gomedia.MP4_FLAG_KEYFRAME))
	if false {
		videoTrackId := m.AddVideoTrack(mp4gomedia.MP4_CODEC_H264, mp4gomedia.WithExtraData(nil))
		audioTrackId := m.AddAudioTrack(mp4gomedia.MP4_CODEC_G711A)
		_ = m.Write(videoTrackId, nil, 1, 1)
		_ = m.Write(audioTrackId, nil, 2, 2)
		_ = m.WriteTrailer()
	}

	return &Mp4Muxer{m: m}, err
}

func testMP4() {
	inFilename := `..\container\res\UA_555.mkv`
	fpInput, _ := os.Open(inFilename)
	defer fpInput.Close()

	var (
		demuxer = mkv.NewDemuxer(fpInput)
		streams []av.CodecData
	)

	if streamList, err := demuxer.Streams(); err == nil {
		for _, stream := range streamList {
			switch s := stream.(type) {
			case h264parser.CodecData:
				if v, err := h264parser.NewCodecDataFromSPSAndPPS(s.SPS(), s.PPS()); err != nil {
					log.Println(err)
					return
				} else {
					streams = append(streams, v)
				}
			case h265parser.CodecData:
				if v, err := h265parser.NewCodecDataFromVPSAndSPSAndPPS(s.VPS(), s.SPS(), s.PPS()); err != nil {
					log.Println(err)
					return
				} else {
					streams = append(streams, v)
				}
			case aacparser.CodecData:
				if v, err := aacparser.NewCodecDataFromMPEG4AudioConfig(s.Config); err != nil {
					log.Println(err)
					return
				} else {
					streams = append(streams, v)
				}
			case opusparser.CodecData:
				v := opusparser.NewCodecData(s.Channels)
				streams = append(streams, v)
			}
		}
	}

	fp, _ := os.Create("out.mp4")
	defer fp.Close()

	muxer := MP4VDKMuxer(fp)
	if err := muxer.WriteHeader(streams); err != nil {
		log.Println(err)
		return
	}

	var t time.Duration
	for {
		pkt, err := demuxer.ReadPacket()
		if err != nil {
			break
		}

		pkt.Time = t
		if err := muxer.WritePacket(pkt); err != nil {
			break
		}

		t += 40 * time.Millisecond
	}

	if err := muxer.WriteTrailer(); err != nil {
		log.Println(err)
	}
}

func testMp4go2rtc() {
	var (
		sps       []byte
		pps       []byte
		codecList []*core.Codec
	)

	codecList = append(codecList, &core.Codec{
		Name:        core.CodecH264,
		ClockRate:   90000,
		FmtpLine:    codecs.NewH264Param(sps, pps).GetFmtpString(),
		PayloadType: core.PayloadTypeRAW,
	})

	m := mp4go2rtc.Muxer{}
	_, _ = m.GetInit(codecList)
	m.Marshal(0, nil)
}

func TestMp4ff() {
	objectFile, err := mp4ffmp4.ReadMP4File(`out4.mp4`)
	if err != nil {
		return
	}

	_ = objectFile
}

func testMp4GoMedia() {
	w, _ := mp4gomedia.CreateMp4Muxer(nil, mp4gomedia.WithMp4Flag(mp4gomedia.MP4_FLAG_FRAGMENT))
	videoTrack := w.AddVideoTrack(mp4gomedia.MP4_CODEC_H264, mp4gomedia.WithExtraData(nil))
	audioTrack := w.AddAudioTrack(mp4gomedia.MP4_CODEC_G711A)
	_ = w.Write(videoTrack, nil, 0, 0)
	_ = w.Write(audioTrack, nil, 0, 0)
	_ = w.WriteTrailer()
}
