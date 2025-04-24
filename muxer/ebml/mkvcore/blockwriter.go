// Copyright 2019 The ebml-go authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mkvcore

import (
	"errors"
	"io"
	"math"
	"sync"
	"time"

	"github.com/greendrake/cctv/muxer/ebml"
	"github.com/greendrake/cctv/muxer/ebml/core"
)

// ErrIgnoreOldFrame means that a frame has too old timestamp and ignored.
var ErrIgnoreOldFrame = errors.New("too old frame")

type blockWriter struct {
	trackNumber uint64
	f           chan *frame
	wg          *sync.WaitGroup
	fin         chan struct{}
}

type frame struct {
	trackNumber uint64
	keyframe    bool
	timestamp   int64
	b           []byte
}

func (w *blockWriter) Write(keyframe bool, timestamp int64, b []byte) (int, error) {
	w.f <- &frame{
		trackNumber: w.trackNumber,
		keyframe:    keyframe,
		timestamp:   timestamp,
		b:           b,
	}
	return len(b), nil
}

func (w *blockWriter) Close() error {
	w.wg.Done()

	// If it is the last writer, block until closing output writer.
	w.fin <- struct{}{}

	return nil
}

// TrackDescription stores track number and its TrackEntry struct.
type TrackDescription struct {
	TrackNumber uint64
	TrackEntry  core.TrackEntry
}

// NewSimpleBlockWriter creates BlockWriteCloser for each track specified as tracks argument.
// Blocks will be written to the writer as EBML SimpleBlocks.
// Given io.WriteCloser will be closed automatically; don't close it by yourself.
// Frames written to each track must be sorted by their timestamp.
func NewSimpleBlockWriter(w0 io.WriteCloser, tracks []TrackDescription, opts ...BlockWriterOption) ([]BlockWriteCloser, error) {
	options := &BlockWriterOptions{
		BlockReadWriterOptions: BlockReadWriterOptions{
			onFatal: func(err error) {
				panic(err)
			},
		},
		ebmlHeader:  nil,
		segmentInfo: nil,
		interceptor: nil,
		seekHead:    false,
	}
	for _, o := range opts {
		if err := o.ApplyToBlockWriterOptions(options); err != nil {
			return nil, err
		}
	}

	var (
		offsetSeekHeader int
		offsetCluster    int
	)

	w := &writerWithSizeCount{w: w0}

	// w *writerWithSizeCount, tracks []TrackDescription, options *BlockWriterOptions,
	if err := writeHeader(w, tracks, options, nil, nil, nil, func(offSeek int, offCluster int) {
		offsetSeekHeader = offSeek
		offsetCluster = offCluster
	}); err != nil {
		return nil, err
	}

	ch := make(chan *frame, 10000)
	fin := make(chan struct{}, len(tracks)-1)
	wg := sync.WaitGroup{}
	var ws []BlockWriteCloser
	var fw []BlockWriter
	var fr []BlockReader

	for _, t := range tracks {
		wg.Add(1)
		var chSrc chan *frame
		if options.interceptor == nil {
			chSrc = ch
		} else {
			chSrc = make(chan *frame)
			fr = append(fr, &filterReader{chSrc})
			fw = append(fw, &filterWriter{t.TrackNumber, ch})
		}
		ws = append(ws, &blockWriter{
			trackNumber: t.TrackNumber,
			f:           chSrc,
			wg:          &wg,
			fin:         fin,
		})
	}

	filterFlushed := make(chan struct{})
	if options.interceptor != nil {
		go func() {
			options.interceptor.Intercept(fr, fw)
			close(filterFlushed)
		}()
	} else {
		close(filterFlushed)
	}

	closed := make(chan struct{})
	go func() {
		wg.Wait()
		for _, c := range fr {
			c.(*filterReader).close()
		}
		<-filterFlushed
		close(closed)
	}()

	writeTrack(offsetSeekHeader, offsetCluster, w, tracks, ch, fin, closed, options)

	return ws, nil
}

func writeHeader(w *writerWithSizeCount, tracks []TrackDescription, options *BlockWriterOptions,
	clusterPos, cuesPos *uint64, fileDuration *float64,
	cb func(offsetSeekHeader int, clusterOffset int)) error {
	var (
		offsetSeekHeader int
		offsetCluster    int
	)

	{
		// EBML header
		header := myFlexHeaderHeader{
			Header: options.ebmlHeader,
		}

		if err := ebml.Marshal(&header, w, options.marshalOpts...); err != nil {
			return err
		}
		offsetSeekHeader = w.Size() + 12
	}
	{
		// Segment(SeekHead/Info/Tracks/Tag)
		header := myFlexHeaderSegment{
			Segment: flexSegment{
				SeekHead: nil,
				Info:     options.segmentInfo,
				Tracks:   flexTracks{},
				Tags:     nil,
				Cluster:  nil,
			},
		}
		if len(options.tags) > 0 {
			header.Segment.Tags = append(header.Segment.Tags, Tags{
				Tag: options.tags,
			})
		}

		type SegmentInfo interface {
			SetDuration(duration float64)
			SetDateUTC(date time.Time)
		}

		if info, ok := header.Segment.Info.(SegmentInfo); ok {
			if fileDuration != nil {
				info.SetDuration(*fileDuration)
			} else {
				info.SetDuration(5)
				info.SetDateUTC(time.Now().Truncate(time.Millisecond))
			}
		}

		for _, t := range tracks {
			track := t
			header.Segment.Tracks.TrackEntry = append(header.Segment.Tracks.TrackEntry, track.TrackEntry)
		}

		if options.seekHead {
			if err := setSeekHead2(&header, clusterPos, cuesPos, options); err != nil {
				return err
			}
		}

		if err := ebml.Marshal(&header, w, options.marshalOpts...); err != nil {
			return err
		}

		// EBML void
		{
			var void = Void{
				Void: make([]byte, 160),
			}

			if err := ebml.Marshal(&void, w, options.marshalOpts...); err != nil {
				return err
			}
		}

		offsetCluster = w.Size()
		w.Clear()
	}

	if cb != nil {
		cb(offsetSeekHeader, offsetCluster)
	}

	return nil
}

// Cues represents Cues element struct.
type Cues struct {
	CuePoint []CuePoint `ebml:"CuePoint"`
}

// CuePoint represents CuePoint element struct.
type CuePoint struct {
	CueTime           uint64             `ebml:"CueTime"`
	CueTrackPositions []CueTrackPosition `ebml:"CueTrackPositions"`
}

// CueTrackPosition represents CueTrackPosition element struct.
type CueTrackPosition struct {
	CueTrack           uint64 `ebml:"CueTrack"`
	CueClusterPosition uint64 `ebml:"CueClusterPosition"`
	CueBlockNumber     uint64 `ebml:"CueBlockNumber,omitempty"`
}

func writeTrack(offsetSeekHeader int, clusterOffset int, w *writerWithSizeCount, tracks []TrackDescription, ch chan *frame, fin chan struct{}, closed chan struct{}, options *BlockWriterOptions) {
	tNextCluster := math.MaxInt16 - options.maxKeyframeInterval

	var (
		clusterPosition uint64 = uint64(clusterOffset)
		cueBlockNumber  uint64
	)

	type CustomCluster struct {
		Cluster simpleBlockCluster `ebml:"Cluster"`
	}

	var fn = func() {
		const invalidTimestamp = int64(0x7FFFFFFFFFFFFFFF)
		tc0 := invalidTimestamp
		tc1 := invalidTimestamp
		lastTc := int64(0)

		cues := struct {
			Cues *Cues `ebml:"Cues"`
		}{
			Cues: &Cues{},
		}

		var cluster *CustomCluster

		defer func() {
			// Finalize WebM
			if cluster == nil {
				// 最少一个cluster
				if tc0 == invalidTimestamp {
					// No data written
					tc0 = 0
				}

				cluster = &CustomCluster{
					Cluster: simpleBlockCluster{
						Timestamp: uint64(lastTc - tc0),
						PrevSize:  uint64(w.Size()),
					},
				}

				if err := ebml.Marshal(cluster, w, options.marshalOpts...); err != nil {
					if options.onFatal != nil {
						options.onFatal(err)
					}
				}
			}

			clusterPosition += uint64(w.Size())

			// 索引
			if options.cues {
				if err := ebml.Marshal(&cues, w, options.marshalOpts...); err != nil {
					if options.onFatal != nil {
						options.onFatal(err)
					}
				}
			}

			// 更新头部信息, 文件时长/定位等
			if seeker, ok := w.w.(io.Seeker); ok {
				if _, err := seeker.Seek(0, io.SeekStart); err == nil {
					var (
						clusterPos   = uint64(clusterOffset) - uint64(offsetSeekHeader)
						cuesPos      = uint64(clusterPosition) - uint64(offsetSeekHeader)
						fileDuration = float64(lastTc)
					)

					w.Clear()

					for i := 0; i < len(tracks); i++ {
						// tracks[i].TrackEntry.SetAudioSamplingFrequency(8000)
					}
					_ = writeHeader(w, tracks, options, &clusterPos, &cuesPos, &fileDuration, nil)
				}
			}

			_ = w.Close()
			<-fin // read one data to release blocked Close()
			select {
			case <-fin:
				return
			case <-ch:
				// ignore
			}
		}()

		defer func() {
			if cluster == nil {
				return
			}

			w.Clear()
			if err := ebml.Marshal(cluster, w, options.marshalOpts...); err != nil {
				if options.onFatal != nil {
					options.onFatal(err)
				}
				return
			}

			clusterSize := uint64(w.Size())
			clusterPosition += clusterSize
		}()

		var onFrame = func(f *frame) {
			if tc0 == invalidTimestamp {
				tc0 = f.timestamp
			}

			lastTc = f.timestamp
			tc := f.timestamp - tc1
			if tc1 == invalidTimestamp || tc >= math.MaxInt16 ||
				(f.trackNumber == options.mainTrackNumber && tc >= tNextCluster && f.keyframe) {
				// Create new Cluster
				tc1 = f.timestamp
				tc = 0
				cueBlockNumber = 0

				w.Clear()
				if cluster != nil {
					if err := ebml.Marshal(cluster, w, options.marshalOpts...); err != nil {
						if options.onFatal != nil {
							options.onFatal(err)
						}
						return
					}
				}
				clusterSize := uint64(w.Size())
				clusterPosition += clusterSize

				cluster = &CustomCluster{
					Cluster: simpleBlockCluster{
						Timestamp: uint64(tc1 - tc0),
						PrevSize:  uint64(clusterSize),
					},
				}
			}

			if tc <= -math.MaxInt16 {
				// Ignore too old frame
				if options.onError != nil {
					options.onError(ErrIgnoreOldFrame)
				}
				return
			}

			cluster.Cluster.SimpleBlock = append(cluster.Cluster.SimpleBlock, ebml.Block{
				TrackNumber: f.trackNumber,
				Timecode:    int16(tc),
				Keyframe:    f.keyframe,
				Data:        [][]byte{f.b},
			})
			cueBlockNumber++

			if options.cues && cueBlockNumber == 1 && f.keyframe {
				cues.Cues.CuePoint = append(cues.Cues.CuePoint, CuePoint{
					CueTime: uint64(tc1),
					CueTrackPositions: []CueTrackPosition{
						{
							CueTrack:           f.trackNumber,
							CueClusterPosition: clusterPosition - uint64(offsetSeekHeader),
							CueBlockNumber:     cueBlockNumber,
						},
					},
				})
			}
		}

		for {
			select {
			case <-closed:
				for {
					select {
					case f := <-ch:
						onFrame(f)
					default:
						return
					}
				}
			case f := <-ch:
				if tc0 == invalidTimestamp {
					tc0 = f.timestamp
				}

				lastTc = f.timestamp
				tc := f.timestamp - tc1
				if tc1 == invalidTimestamp || tc >= math.MaxInt16 ||
					(f.trackNumber == options.mainTrackNumber && tc >= tNextCluster && f.keyframe) {
					// Create new Cluster
					tc1 = f.timestamp
					tc = 0
					cueBlockNumber = 0

					w.Clear()
					if cluster != nil {
						if err := ebml.Marshal(cluster, w, options.marshalOpts...); err != nil {
							if options.onFatal != nil {
								options.onFatal(err)
							}
							return
						}
					}
					clusterSize := uint64(w.Size())
					clusterPosition += clusterSize

					cluster = &CustomCluster{
						Cluster: simpleBlockCluster{
							Timestamp: uint64(tc1 - tc0),
							PrevSize:  uint64(clusterSize),
						},
					}
				}

				if tc <= -math.MaxInt16 {
					// Ignore too old frame
					if options.onError != nil {
						options.onError(ErrIgnoreOldFrame)
					}
					return
				}

				cluster.Cluster.SimpleBlock = append(cluster.Cluster.SimpleBlock, ebml.Block{
					TrackNumber: f.trackNumber,
					Timecode:    int16(tc),
					Keyframe:    f.keyframe,
					Data:        [][]byte{f.b},
				})
				cueBlockNumber++

				if options.cues && cueBlockNumber == 1 && f.keyframe {
					cues.Cues.CuePoint = append(cues.Cues.CuePoint, CuePoint{
						CueTime: uint64(tc1),
						CueTrackPositions: []CueTrackPosition{
							{
								CueTrack:           f.trackNumber,
								CueClusterPosition: clusterPosition - uint64(offsetSeekHeader),
								CueBlockNumber:     cueBlockNumber,
							},
						},
					})
				}
			}
		}
	}

	go fn()
}
