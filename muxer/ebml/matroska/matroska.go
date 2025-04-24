package matroska

import (
	"fmt"
	"io"
	// "log"
	"math"
	"time"

	"github.com/greendrake/cctv/muxer/ebml/mkv"
	"github.com/greendrake/cctv/muxer/ebml/mkvcore"
	"github.com/greendrake/cctv/muxer/ebml/webm"
)

// https://www.matroska.org/technical/elements.html
// https://www.webmproject.org/docs/container/

type WriteSeekCloser interface {
	io.Writer
	io.Seeker
	io.Closer
}

type Matroska struct {
	w      *writerFileSize
	tracks []Track
}

func Open(w WriteSeekCloser, tracks ...Track) (*Matroska, error) {
	if len(tracks) == 0 {
		return nil, ErrNoTracks
	}

	m := &Matroska{
		w: &writerFileSize{
			w: w,
		},
		tracks: tracks,
	}

	m.modifyTrackNumber()

	if err := m.open(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Matroska) open() error {

	var (
		w         = m.w
		wemTracks []*webm.TrackEntry
		opts      = m.getOptions()
	)

	for _, track := range m.tracks {
		t := track.GetTrackEntry()
		wemTracks = append(wemTracks, t)
	}

	writers, err := webm.NewSimpleBlockWriter(w, wemTracks, opts...)
	if err != nil {
		return err
	}

	if len(writers) == 0 || len(writers) != len(m.tracks) {
		return fmt.Errorf("open error. error writers  %v %v", len(writers), len(m.tracks))
	}

	for i := 0; i < len(m.tracks); i++ {
		m.tracks[i].setBlockWriteCloser(writers[i])
	}

	return nil
}

func (m *Matroska) Close() {
	for _, track := range m.tracks {
		_ = track.GetWriter().Close()
	}
	m.w.Close()
}

func (m *Matroska) FileSize() int {
	return m.w.FileSize()
}

func (m *Matroska) GetTracks() []Track {
	return m.tracks
}

func (m *Matroska) WriteTrack(t Track, timestamp time.Duration, b []byte, keyframe ...bool) (int, error) {
	return t.Write(timestamp, b, keyframe...)
}

func (m *Matroska) WriteVideo(timestamp time.Duration, b []byte) (int, error) {
	for _, track := range m.tracks {
		if track.IsVideo() {
			return track.Write(timestamp, b)
		}
	}
	return -1, ErrNotFoundTrack
}

func (m *Matroska) WriteAudio(timestamp time.Duration, b []byte) (int, error) {
	for _, track := range m.tracks {
		if track.IsAudio() {
			return track.Write(timestamp, b)
		}
	}
	return -1, ErrNotFoundTrack
}

func (m *Matroska) getOptions() []mkvcore.BlockWriterOption {
	var opts = []mkvcore.BlockWriterOption{
		mkvcore.WithSeekHead(true),
		mkvcore.WithCues(true),
		mkvcore.WithEBMLHeader(mkv.DefaultEBMLHeader),
		mkvcore.WithSegmentInfo(mkv.DefaultSegmentInfo),
		mkvcore.WithOnErrorHandler(func(err error) {
			panic(err)
		}),
		mkvcore.WithOnFatalHandler(func(err error) {
			panic(err)
		}),
		//mkvcore.WithMarshalOptions(ebml.WithDataSizeLen(2)),
	}

	if len(m.tracks) == 1 && m.tracks[0].IsAudio() {
		// only audio
		opts = append(opts, mkvcore.WithMaxKeyframeInterval(1, math.MaxInt16-5000))
	} else {
		opts = append(opts, mkvcore.WithMaxKeyframeInterval(1, 900*0x6FFF))
	}

	return opts
}

func (m *Matroska) modifyTrackNumber() {
	var (
		videoTrack Track
		otherTrack []Track
	)

	for _, t := range m.tracks {
		track := t
		if videoTrack == nil && t.IsVideo() {
			videoTrack = track
		} else {
			otherTrack = append(otherTrack, track)
		}
	}

	var trackNumber = 1
	if videoTrack != nil {
		videoTrack.setTrackNumber(trackNumber)
		trackNumber++
	}

	for _, track := range otherTrack {
		track.setTrackNumber(trackNumber)
		trackNumber++
	}
}
