package formats

import (
	"fmt"
	"io"

	"gitee.com/general252/go-wav"
)

type WavReader interface {
	io.Reader
	io.ReaderAt
}

func EmitWavReader(r WavReader, emit func(format *wav.WavFormat, data []byte)) error {
	return EmitWavReaderExt(r, func(format *wav.WavFormat) int {
		return int(format.NumChannels) * int(format.SampleRate) / 25
	}, emit)
}

func EmitWavReaderExt(r WavReader, getBuffSize func(format *wav.WavFormat) int, emit func(format *wav.WavFormat, data []byte)) error {
	reader := wav.NewReader(r)

	format, err := reader.Format()
	if err != nil {
		return err
	}

	if format.AudioFormat != wav.AudioFormatALaw {
		return fmt.Errorf("not pcma")
	}

	bufSize := getBuffSize(format)
	for {
		buff := make([]byte, bufSize)
		if n, err := reader.Read(buff); err != nil {
			if err == io.EOF {
				break
			}
			return err
		} else {
			emit(format, buff[:n])
		}
	}

	return nil
}
