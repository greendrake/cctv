package matroska

import "errors"

var (
	ErrNoBlockWriter  = errors.New("no black writer")
	ErrNotFoundTrack  = errors.New("no found track")
	ErrNoTracks       = errors.New("no tracks")
	ErrH264PacketSize = errors.New("packet size error")
	ErrNotAnnexBData  = errors.New("not AnnexB format data")
)
