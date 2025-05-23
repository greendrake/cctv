package bits

type Reader struct {
	EOF  bool   // if end of buffer raised during reading
	buf  []byte // total buf
	byte byte   // current byte
	bits byte   // bits left in byte
	pos  int    // current pos in buf
}

func NewReader(b []byte) *Reader {
	return &Reader{buf: b}
}

//goland:noinspection GoStandardMethods
func (r *Reader) ReadByte() byte {
	if r.bits != 0 {
		return r.ReadBits8(8)
	}

	if r.pos >= len(r.buf) {
		r.EOF = true
		return 0
	}

	b := r.buf[r.pos]
	r.pos++
	return b
}

func (r *Reader) ReadBit() byte {
	if r.bits == 0 {
		r.byte = r.ReadByte()
		r.bits = 7
	} else {
		r.bits--
	}

	return (r.byte >> r.bits) & 0b1
}

func (r *Reader) ReadBits(n byte) (res uint32) {
	for i := n - 1; i != 255; i-- {
		res |= uint32(r.ReadBit()) << i
	}
	return
}

func (r *Reader) ReadBits8(n byte) (res uint8) {
	for i := n - 1; i != 255; i-- {
		res |= r.ReadBit() << i
	}
	return
}

func (r *Reader) ReadBits64(n byte) (res uint64) {
	for i := n - 1; i != 255; i-- {
		res |= uint64(r.ReadBit()) << i
	}
	return
}

// ReadUEGolomb - ReadExponentialGolomb (unsigned)
func (r *Reader) ReadUEGolomb() uint32 {
	var size byte
	for size = 0; size < 32; size++ {
		if b := r.ReadBit(); b != 0 || r.EOF {
			break
		}
	}
	return r.ReadBits(size) + (1 << size) - 1
}
