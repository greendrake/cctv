package frame

import (
	"fmt"
	"time"
)

type HeaderCommon struct {
	// First 3 bytes are always the same:  0x00, 0x00, 0x01
	B1   byte
	B2   byte
	B3   byte
	Type Type
}

func (h HeaderCommon) GetHumanType() string {
	return THMap[h.Type]
}

type Header interface {
	GetOwnLength() uint8
	GetLength() uint32
	GetMediaType(ft Type) string
}

type DateTimeHolder interface {
	GetDateTime() time.Time
}

type Length[T uint16 | uint32] struct {
	Length T
}

func (l Length[T]) GetLength() uint32 {
	return uint32(l.Length)
}

type DateTime struct {
	DateTime uint32
}

type Misc[T byte | int32] struct {
	Misc T
}

func (misc Misc[T]) GetMediaType(ft Type) string {
	last4Bits := misc.Misc & 0xF
	mediaType := "unknown"
	switch ft {
	case T_VideoI:
		switch last4Bits {
		case 0x01:
			mediaType = "MPEG4"
		case 0x02:
			mediaType = "H264"
		case 0x03:
			mediaType = "H265"
		}
	case T_Audio:
		switch last4Bits {
		case 0x0E:
			mediaType = "G711A"
		}
	case T_Picture:
		switch last4Bits {
		case 0x00:
			mediaType = "JPEG"
		}
	case T_Info:
		switch last4Bits {
		case 0x01:
			mediaType = "DeviceInfo"
		}
	}
	return mediaType
}

func (dt DateTime) GetDateTime() time.Time {
	second := int(dt.DateTime & 0x3F)
	minute := int((dt.DateTime & 0xFC0) >> 6)
	hour := int((dt.DateTime & 0x1F000) >> 12)
	day := int((dt.DateTime & 0x3E0000) >> 17)
	month := int((dt.DateTime & 0x3C00000) >> 22)
	year := int(((dt.DateTime & 0xFC000000) >> 26) + 2000)
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
}

type HeaderVideoI struct {
	Misc[int32]
	DateTime
	Length[uint32]
}

func (f HeaderVideoI) GetOwnLength() uint8 {
	return 16
}

func printBits(n int32) {
	for i := 31; i >= 0; i-- {
		bit := (n >> uint(i)) & 1
		fmt.Printf("%d", bit)
		if i%8 == 0 {
			fmt.Printf(" ") // add a space every 8 bits for better readability
		}
	}
	fmt.Println()
}
func joinBytes(byte1, byte2 byte) uint16 {
	return uint16(byte1)<<8 | uint16(byte2)
}
func (videoI HeaderVideoI) GetWidth() uint16 {
	return joinBytes(byte(videoI.Misc.Misc)&0b00110000>>4, byte(videoI.Misc.Misc>>16)) * 8
}
func (videoI HeaderVideoI) GetHeight() uint16 {
	return joinBytes(byte(videoI.Misc.Misc)&0b11000000>>6, byte(videoI.Misc.Misc>>24)) * 8
}
func (videoI HeaderVideoI) GetFPS() uint8 {
	return uint8(videoI.Misc.Misc >> 8)
}

type HeaderVideoP struct {
	Length[uint32]
}

func (videoP HeaderVideoP) GetOwnLength() uint8 {
	return 8
}
func (videoP HeaderVideoP) GetMediaType(ft Type) string {
	// This method should never be called. It exists only to satisfy the interface.
	return "unknown video type"
}

type HeaderAudio struct {
	Misc[byte]
	SampleRate byte
	Length[uint16]
}

func (f HeaderAudio) GetOwnLength() uint8 {
	return 8
}

type HeaderPicture struct {
	Misc[int32]
	DateTime
	Length[uint32]
}

func (f HeaderPicture) GetOwnLength() uint8 {
	return 16
}

type HeaderInfo struct {
	Misc[byte]
	_ byte // unused
	Length[uint16]
}

func (f HeaderInfo) GetOwnLength() uint8 {
	return 8
}
