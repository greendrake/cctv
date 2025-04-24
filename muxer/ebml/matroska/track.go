package matroska

import (
	"sync/atomic"
	"time"

	"github.com/greendrake/cctv/muxer/ebml/core"
	"github.com/greendrake/cctv/muxer/ebml/webm"
)

type Track interface {
	IsVideo() bool
	IsAudio() bool
	GetTrackEntry() *webm.TrackEntry
	Write(timestamp time.Duration, b []byte, keyframe ...bool) (int, error)

	GetWriter() webm.BlockWriteCloser
	setBlockWriteCloser(blockWriter webm.BlockWriteCloser)

	setTrackNumber(trackNumber int)

	mustEmbedUnimplemented()
}

type UnimplementedTrack struct {
	track       webm.TrackEntry
	blockWriter webm.BlockWriteCloser
}

func (c *UnimplementedTrack) GetTrackEntry() *webm.TrackEntry {
	return &c.track
}

func (c *UnimplementedTrack) IsVideo() bool {
	return c.track.TrackType == core.TrackTypeVideo
}

func (c *UnimplementedTrack) IsAudio() bool {
	return c.track.TrackType == core.TrackTypeAudio
}

func (c *UnimplementedTrack) setTrackNumber(trackNumber int) {
	c.track.TrackNumber = uint64(trackNumber)
}

func (c *UnimplementedTrack) setBlockWriteCloser(blockWriter webm.BlockWriteCloser) {
	c.blockWriter = blockWriter
}

func (c *UnimplementedTrack) GetWriter() webm.BlockWriteCloser {
	return c.blockWriter
}

func (c *UnimplementedTrack) Write(timestamp time.Duration, b []byte, keyframe ...bool) (int, error) {
	if c.blockWriter == nil {
		return 0, ErrNoBlockWriter
	}

	key := false
	if len(keyframe) > 0 {
		key = keyframe[0]
	}

	return c.blockWriter.Write(key, timestamp.Milliseconds(), b)
}

func (c *UnimplementedTrack) mustEmbedUnimplemented() {}

var trackUid = uint64(100)

func getTrackUID() uint64 {
	return atomic.AddUint64(&trackUid, 1)
}
