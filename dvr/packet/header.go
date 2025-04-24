package packet

// import (
//     "fmt"
// )

// There are 2 types of packets:
// 1. For control messages.
// 2. For media messages.
// Both have a header 20 bytes long, structured almost identical. Only the 2 fields differ:

type Header struct {
	HeadFlag       byte // Always 0xFF
	Version        byte
	_              byte // Reserved
	_              byte // Reserved
	SessionId      int32
	SequenceNumber uint32 // Packet ordinal number, starting from 0 and re-setting back to 0 when reached the max value
	// Control messages: total packets in multi-packet message. 0 and 1 mean the same (1), more than 1 indicates how many
	// Media messages: Channel
	TotalPacketsOrChannel uint8
	// Control messages: current packet number (for multi-packet messages)
	// Media messages: 0x01 means end of data, otherwise 0x0
	CurrentPacketOrEndFlag uint8
	Code                   Code
	DataLength             uint32 // For control messages, 16K max
}
