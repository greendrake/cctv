package message

import "github.com/greendrake/cctv/dvr/packet"

type Message struct {
	Code packet.Code
	Data []byte
}
