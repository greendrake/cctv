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

// Package webm provides the WebM multimedia writer.
//
// The package implements block data writer for multi-track WebM container.
package webm

import (
	"bytes"
	"log"
	"time"

	"github.com/greendrake/cctv/muxer/ebml"
	"github.com/greendrake/cctv/muxer/ebml/core"
)

// https://www.matroska.org/technical/elements.html
// https://www.matroska.org/technical/diagram.html
// https://github.com/ebml-go/webm/blob/master/parser.go
// https://code.google.com/archive/p/ebml-go/source/default/source (ebml-go/webm/parser.go)

// EBMLHeader represents EBML header struct.
type EBMLHeader struct {
	EBMLVersion        uint64 `ebml:"EBMLVersion"`
	EBMLReadVersion    uint64 `ebml:"EBMLReadVersion"`
	EBMLMaxIDLength    uint64 `ebml:"EBMLMaxIDLength"`
	EBMLMaxSizeLength  uint64 `ebml:"EBMLMaxSizeLength"`
	DocType            string `ebml:"EBMLDocType"`
	DocTypeVersion     uint64 `ebml:"EBMLDocTypeVersion"`
	DocTypeReadVersion uint64 `ebml:"EBMLDocTypeReadVersion"`
}

// Seek represents Seek element struct.
type Seek struct {
	SeekID       []byte `ebml:"SeekID"`
	SeekPosition uint64 `ebml:"SeekPosition"`
}

// SeekHead represents SeekHead element struct.
type SeekHead struct {
	Seek []Seek `ebml:"Seek"`
}

// Info represents Info element struct.
type Info struct {
	TimecodeScale uint64    `ebml:"TimecodeScale"`
	MuxingApp     string    `ebml:"MuxingApp,omitempty"`
	WritingApp    string    `ebml:"WritingApp,omitempty"`
	Duration      float64   `ebml:"Duration,omitempty"`
	DateUTC       time.Time `ebml:"DateUTC,omitempty"`
}

func (c *Info) GetDuration() time.Duration {
	return time.Nanosecond * time.Duration(c.Duration*float64(c.TimecodeScale))
}

func (c *Info) SetDuration(duration float64) {
	c.Duration = duration
}

func (c *Info) SetDateUTC(date time.Time) {
	c.DateUTC = date
}

// TrackEntry represents TrackEntry element struct.
type TrackEntry struct {
	Name            string         `ebml:"Name,omitempty"`
	TrackNumber     uint64         `ebml:"TrackNumber"`
	TrackUID        uint64         `ebml:"TrackUID"`
	CodecID         core.CodecType `ebml:"CodecID"`
	CodecDelay      uint64         `ebml:"CodecDelay,omitempty"`
	TrackType       core.TrackType `ebml:"TrackType"`
	DefaultDuration uint64         `ebml:"DefaultDuration,omitempty"`
	SeekPreRoll     uint64         `ebml:"SeekPreRoll,omitempty"`
	Audio           *Audio         `ebml:"Audio"`
	Video           *Video         `ebml:"Video"`
	CodecPrivate    []byte         `ebml:"CodecPrivate,omitempty"`
	Void            []byte         `ebml:"Void,omitempty"`
}

func (c *TrackEntry) SetAudioSamplingFrequency(samplingFrequency float64) {
	if c == nil || c.Audio == nil {
		return
	}

	c.Audio.SamplingFrequency = samplingFrequency
}

func (c *TrackEntry) SetCodecPrivate(codecPrivate []byte) error {

	type TempTrackEntry struct {
		CodecPrivate []byte `ebml:"CodecPrivate,omitempty"`
		Void         []byte `ebml:"Void,omitempty"`
	}
	var tmp = &TempTrackEntry{
		CodecPrivate: c.CodecPrivate,
		Void:         c.Void,
	}

	var buff = bytes.Buffer{}

	// 原数据大小
	_ = ebml.Marshal(tmp, &buff)
	totalSize := buff.Len()

	tmp.CodecPrivate = codecPrivate
	tmp.Void = nil

	// 新数据(除Void外)大小
	buff.Reset()
	_ = ebml.Marshal(tmp, &buff)
	newSize := buff.Len()

	c.CodecPrivate = codecPrivate

	ebmlIdSize := 1
	if t, err := ebml.ElementTypeFromString("Void"); err == nil {
		ebmlIdSize = len(t.Bytes())
	}
	// 新的Void大小
	voidSize := totalSize - newSize - ebmlIdSize - ebml.ElementVoidSize // ebml id, ebml length
	c.Void = make([]byte, voidSize)
	return nil
}

// Audio represents Audio element struct.
type Audio struct {
	SamplingFrequency       float64 `ebml:"SamplingFrequency"`
	Channels                uint64  `ebml:"Channels"`
	OutputSamplingFrequency float64 `ebml:"OutputSamplingFrequency,omitempty"`
	BitDepth                uint64  `ebml:"BitDepth,omitempty"`
}

// Video represents Video element struct.
type Video struct {
	PixelWidth  uint64 `ebml:"PixelWidth"`
	PixelHeight uint64 `ebml:"PixelHeight"`
	Void        []byte `ebml:"Void,omitempty"`
}

func (v *Video) Set(w uint64, h uint64) {
	var buff = bytes.Buffer{}

	_ = ebml.Marshal(v, &buff)
	totalSize := buff.Len()

	v.PixelWidth = w
	v.PixelHeight = h
	v.Void = nil

	buff.Reset()
	_ = ebml.Marshal(v, &buff)
	newSize := buff.Len()

	ebmlIdSize := 1
	if t, err := ebml.ElementTypeFromString("Void"); err == nil {
		ebmlIdSize = len(t.Bytes())
	}
	voidSize := totalSize - newSize - ebmlIdSize - ebml.ElementVoidSize // ebml id, ebml length, data
	if voidSize > 0 {
		v.Void = make([]byte, voidSize)
	} else {
		log.Printf("error %v", voidSize)
	}
}

// Tracks represents Tracks element struct.
type Tracks struct {
	TrackEntry []TrackEntry `ebml:"TrackEntry"`
}

// BlockGroup represents BlockGroup element struct.
type BlockGroup struct {
	BlockDuration  uint64     `ebml:"BlockDuration,omitempty"`
	ReferenceBlock int64      `ebml:"ReferenceBlock,omitempty"`
	Block          ebml.Block `ebml:"Block"`
}

// Cluster represents Cluster element struct.
type Cluster struct {
	Timecode    uint64       `ebml:"Timecode"`
	PrevSize    uint64       `ebml:"PrevSize,omitempty"`
	BlockGroup  []BlockGroup `ebml:"BlockGroup"`
	SimpleBlock []ebml.Block `ebml:"SimpleBlock"`
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

// Segment represents Segment element struct.
type Segment struct {
	SeekHead *SeekHead `ebml:"SeekHead"`
	Info     Info      `ebml:"Info"`
	Tracks   Tracks    `ebml:"Tracks"`
	Cluster  []Cluster `ebml:"Cluster"`
	Cues     *Cues     `ebml:"Cues"`
}

// SegmentStream represents Segment element struct for streaming.
type SegmentStream struct {
	SeekHead *SeekHead `ebml:"SeekHead"`
	Info     Info      `ebml:"Info"`
	Tracks   Tracks    `ebml:"Tracks"`
	Cluster  []Cluster `ebml:"Cluster,size=unknown"`
}
