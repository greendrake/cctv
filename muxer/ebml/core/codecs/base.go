package codecs

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type NALUFormatType int

const (
	NALUFormatNo     NALUFormatType = 0
	NALUFormatAVCC   NALUFormatType = 1 // length
	NALUFormatAnnexB NALUFormatType = 2 // 00 00 00 01 / 00 00 01
)

func GetNALUFormatType(data []byte) (int, NALUFormatType) {
	if len(data) < 4 {
		return 0, NALUFormatNo
	}

	var (
		startCode3 = []byte{0x00, 0x00, 0x01}
		startCode4 = []byte{0x00, 0x00, 0x00, 0x01}
	)

	if 0 == bytes.Compare(startCode4, data[:4]) {
		return 4, NALUFormatAnnexB
	}

	if NALUAVCCFormatValid(data) {
		return 0, NALUFormatAVCC
	}

	if 0 == bytes.Compare(startCode3, data[:3]) {
		return 3, NALUFormatAnnexB
	}

	return 0, NALUFormatNo
}

// NALUAVCCFormatValid valid AVCC format data
func NALUAVCCFormatValid(data []byte) bool {
	var (
		total  = uint32(len(data))
		offset = uint32(0)
	)

	for {
		off := binary.BigEndian.Uint32(data)
		off += 4
		offset += off

		if offset < total {
			data = data[off:]
		} else if offset == total {
			return true
		} else {
			return false
		}
	}
}

func EmitNALUData(data []byte, withStartCode NALUFormatType, emit func(data []byte)) {
	r := bytes.NewReader(data)
	_, tye := GetNALUFormatType(data)
	switch tye {
	case NALUFormatAnnexB:
		EmitNALUReaderAnnexB(r, withStartCode, emit)
	case NALUFormatAVCC:
		_ = EmitNALUReaderAVCC(r, withStartCode, emit)
	}
}

// EmitNALUReaderAnnexB 00 00 00 01
func EmitNALUReaderAnnexB(r io.Reader, withStartCode NALUFormatType, emit func(data []byte)) {
	rr := bufio.NewReader(r)

	var (
		zeroCount = 0
		found     = false
		nalu      *bytes.Buffer
		startCode = []byte{0x00, 0x00, 0x00, 0x01}
	)

	for {
		b, err := rr.ReadByte()
		if err != nil {
			break
		}

		if found {
			_ = nalu.WriteByte(b)
		}

		if b == 0 {
			zeroCount++
			continue
		} else if b == 1 {
			if zeroCount >= 2 {
				// emit
				startCodeCount := zeroCount + 1
				if nalu != nil && nalu.Len() > startCodeCount {
					data := nalu.Bytes()[:nalu.Len()-startCodeCount]
					switch withStartCode {
					case NALUFormatNo:
					case NALUFormatAVCC:
						buffLength := make([]byte, 4)
						binary.BigEndian.PutUint32(buffLength, uint32(len(data)))
						data = append(buffLength, data...)
					case NALUFormatAnnexB:
						data = append(startCode, data...)
					}
					emit(data)
				}

				found = true
				nalu = bytes.NewBuffer(nil)
				continue
			}
		}

		zeroCount = 0
	}

	if nalu != nil && nalu.Len() > 0 {
		data := nalu.Bytes()
		switch withStartCode {
		case NALUFormatNo:
		case NALUFormatAVCC:
			buffLength := make([]byte, 4)
			binary.BigEndian.PutUint32(buffLength, uint32(len(data)))
			data = append(buffLength, data...)
		case NALUFormatAnnexB:
			data = append(startCode, data...)
		}
		emit(data)
	}
}

// EmitNALUReaderAVCC length
func EmitNALUReaderAVCC(r io.Reader, withStartCode NALUFormatType, emit func(data []byte)) error {
	buff := make([]byte, 4)

	var (
		n1  int
		n2  int64
		err error

		count     int
		startCode = []byte{0x00, 0x00, 0x00, 0x01}
	)
	for {
		if n1, err = r.Read(buff); err != nil {
			if err == io.EOF {
				break
			}
			return err
		} else if n1 != 4 {
			return fmt.Errorf("size error. %v != 4", n1)
		}

		count += n1

		dataSize := binary.BigEndian.Uint32(buff)
		annexBuff := &bytes.Buffer{}
		switch withStartCode {
		case NALUFormatNo:
		case NALUFormatAVCC:
			buffLength := make([]byte, 4)
			binary.BigEndian.PutUint32(buffLength, dataSize)
			annexBuff.Write(buffLength)
		case NALUFormatAnnexB:
			annexBuff.Write(startCode)
		}
		if n2, err = io.CopyN(annexBuff, r, int64(dataSize)); err != nil {
			if err == io.EOF {
				break
			}
			return err
		} else if n2 != int64(dataSize) {
			return fmt.Errorf("data size error. %v != %v", n2, dataSize)
		} else {
			count += int(n2)
		}

		emit(annexBuff.Bytes())
	}

	return nil
}

// ConvertAnnexBToAVCC 00 00 00 01 -> length
func ConvertAnnexBToAVCC(annexbReader io.Reader, avccWriter io.Writer) error {
	avcc := avccWriter

	r := annexbReader

	EmitNALUReaderAnnexB(r, NALUFormatAVCC, func(data []byte) {
		if len(data) == 0 {
			return
		}

		_, _ = avcc.Write(data)
	})

	return nil
}

// ConvertAnnexBToAVCCData 00 00 00 01 -> length
func ConvertAnnexBToAVCCData(data []byte) ([]byte, error) {
	var (
		annexbReader = bytes.NewBuffer(data)
		avccWriter   bytes.Buffer
	)

	if err := ConvertAnnexBToAVCC(annexbReader, &avccWriter); err != nil {
		return nil, err
	}

	return avccWriter.Bytes(), nil
}

// ConvertAVCCToAnnexB length -> 00 00 00 01
func ConvertAVCCToAnnexB(avccReader io.Reader, annexBWriter io.Writer) error {
	return EmitNALUReaderAVCC(avccReader, NALUFormatAnnexB, func(data []byte) {
		_, _ = annexBWriter.Write(data)
	})
}

// ConvertAVCCToAnnexBData length -> 00 00 00 01
func ConvertAVCCToAnnexBData(data []byte) ([]byte, error) {
	var (
		avccReader   = bytes.NewBuffer(data)
		annexBWriter bytes.Buffer
	)

	if err := EmitNALUReaderAVCC(avccReader, NALUFormatAnnexB, func(data []byte) {
		_, _ = annexBWriter.Write(data)
	}); err != nil {
		return nil, err
	}

	return annexBWriter.Bytes(), nil
}
