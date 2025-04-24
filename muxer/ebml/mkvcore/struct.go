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
	"github.com/greendrake/cctv/muxer/ebml"
	"github.com/greendrake/cctv/muxer/ebml/core"
)

type simpleBlockGroup struct {
	Block             []ebml.Block `ebml:"Block"`
	ReferencePriority uint64       `ebml:"ReferencePriority"`
}

type simpleBlockCluster struct {
	//Timecode  uint64             `ebml:"Timecode"`
	Timestamp   uint64             `ebml:"Timestamp"`
	PrevSize    uint64             `ebml:"PrevSize,omitempty"`
	SimpleBlock []ebml.Block       `ebml:"SimpleBlock"`
	BlockGroup  []simpleBlockGroup `ebml:"BlockGroup,omitempty"`
}

type seekFixed struct {
	SeekID       []byte  `ebml:"SeekID"`
	SeekPosition *uint64 `ebml:"SeekPosition,size=8"`
}

type seekHeadFixed struct {
	Seek []seekFixed `ebml:"Seek"`
}

type flexTracks struct {
	TrackEntry []core.TrackEntry `ebml:"TrackEntry"`
}

type flexSegment struct {
	SeekHead *seekHeadFixed       `ebml:"SeekHead,omitempty"`
	Info     interface{}          `ebml:"Info"`
	Tracks   flexTracks           `ebml:"Tracks"`
	Tags     []Tags               `ebml:"Tags,omitempty"`
	Cluster  []simpleBlockCluster `ebml:"Cluster,size=unknown"`
}

type flexHeader struct {
	Header  interface{} `ebml:"EBML"`
	Segment flexSegment `ebml:"Segment,size=unknown"`
}

// ///////////////////////////////////////////////////////////////////////////////////////////////
type myFlexHeaderHeader struct {
	Header interface{} `ebml:"EBML"`
}

type myFlexHeaderSegment struct {
	Segment flexSegment `ebml:"Segment,size=unknown"`
}

type Tags struct {
	Tag []Tag `ebml:"Tag"`
}

type Tag struct {
	Targets   Targets     `ebml:"Targets"`
	SimpleTag []SimpleTag `ebml:"SimpleTag"`
}

type Targets struct {
	TargetTypeValue uint64   `ebml:"TargetTypeValue,omitempty"` // 50
	TargetType      string   `ebml:"TargetType,omitempty"`
	TagTrackUID     []uint64 `ebml:"TagTrackUID,omitempty"`
}

type SimpleTag struct {
	TagName     string `ebml:"TagName"`
	TagString   string `ebml:"TagString"`
	TagLanguage string `ebml:"TagLanguage,omitempty"` // und
	TagDefault  uint64 `ebml:"TagDefault,omitempty"`  // 1 (0-1)
}

type Void struct {
	Void []byte `ebml:"Void,omitempty"`
}

// ///////////////////////////////////////////////////////////////////////////////////////////////

// TrackEntry is a TrackEntry struct with all mandatory elements and commonly used elements.
type TrackEntry struct {
	TrackNumber        uint64
	TrackUID           uint64
	TrackType          core.TrackType
	FlagEnabled        uint8
	FlagDefault        uint8
	FlagForced         uint8
	FlagLacing         uint8
	MinCache           uint64
	DefaultDuration    uint64
	MaxBlockAdditionID uint64
	Name               string
	Language           string
	LanguageIETF       string
	CodecID            string
	CodecDecodeAll     uint8
	SeekPreRoll        uint64
}

func (c *TrackEntry) SetAudioSamplingFrequency(samplingFrequency float64) {
}
