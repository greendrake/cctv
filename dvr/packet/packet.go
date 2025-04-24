package packet

import "fmt"

type Packet struct {
	Header Header
	Data   []byte
}

func (p Packet) GetInfoString() string {
	return fmt.Sprintf("SequenceNumber: %d; Code: %d; DataLength: %d", p.Header.SequenceNumber, p.Header.Code, p.Header.DataLength)
}

func (p Packet) IsMedia() bool {
	return p.Header.Code == MONITOR_DATA
}

func (p Packet) IsSingle() bool {
	if p.IsMedia() {
		return p.Header.DataLength < 16384 && p.Header.DataLength != 8192
	} else {
		return p.Header.TotalPacketsOrChannel < 2
	}
}

func (p Packet) GetOrdinal() uint8 {
	return p.Header.CurrentPacketOrEndFlag
}

func (p Packet) GetTotal() uint8 {
	return p.Header.TotalPacketsOrChannel
}

func (p Packet) IsLast() bool {
	if p.IsSingle() {
		return true
	}
	if p.IsMedia() {
		return p.Header.CurrentPacketOrEndFlag == 1
	} else {
		return p.GetTotal() == (p.Header.CurrentPacketOrEndFlag + 1)
	}
}
