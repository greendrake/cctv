package camera

import (
	// "log"
	"errors"
	"fmt"
	dvrframe "github.com/greendrake/cctv/dvr/frame"
	"github.com/greendrake/cctv/frame"
	"github.com/greendrake/cctv/muxer/ebml/matroska"
	"github.com/greendrake/server_client_hierarchy"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const chunkDuration time.Duration = 10 * time.Minute

// This struct is client to Stream

type MKVWriter struct {
	server_client_hierarchy.Node
	dstDir                string
	videoTimePosition     time.Duration
	lastVideoTimePosition time.Duration
	audioTimePosition     time.Duration
	mkv                   *matroska.Matroska
	HasAudio              bool
	FileSuff              string
	IsHEVC                bool
	lastFrameAudio        bool
	lastAudioTimePosition time.Duration
	closeMutex            sync.Mutex
}

func (w *MKVWriter) Init() {
	w.SetPrincipallyClient(true)
	w.SetIChunkHandler(func(chunk any) {
		w.writeFrame(chunk.(*frame.Frame))
	})
	w.On("stop", func(args ...any) {
		w.close()
	})
}

func (w *MKVWriter) close() {
	w.closeMutex.Lock()
	defer w.closeMutex.Unlock()
	if w.mkv != nil {
		w.mkv.Close()
		w.mkv = nil
	}
}

func (w *MKVWriter) writeFrame(f *frame.Frame) error {
	var err error
	if f.IsVideo && f.IsHEVC && !w.IsHEVC {
		w.IsHEVC = true
	}
	if w.mkv == nil {
		if w.mkv, err = w.createMKVFile(); err != nil {
			return err
		}
	}
	if f.IsVideo {
		if f.IsVideoKeyFrame {
			if (w.videoTimePosition + f.Duration) > chunkDuration {
				go w.mkv.Close()
				if w.mkv, err = w.createMKVFile(); err != nil {
					return err
				}
				w.videoTimePosition = 0
				w.audioTimePosition = 0
				w.lastVideoTimePosition = 0
				w.lastAudioTimePosition = 0
				w.lastFrameAudio = false
			}
		}
		_, err = w.mkv.WriteVideo(w.videoTimePosition, *f.Data)
		if err != nil {
			return errors.New(fmt.Sprintf("Error writing video frame at position %s: [%s]. Last video position: %s; Last audio position: %s", w.videoTimePosition, err, w.lastVideoTimePosition, w.lastAudioTimePosition))
		}
		w.lastVideoTimePosition = w.videoTimePosition
		w.videoTimePosition += f.Duration
		if w.lastFrameAudio {
			w.lastFrameAudio = false
		}
	} else if f.IsAudio && w.HasAudio {
		if !w.lastFrameAudio {
			w.audioTimePosition = w.lastVideoTimePosition
		}
		_, err = w.mkv.WriteAudio(w.audioTimePosition, *f.Data)
		if err != nil {
			return errors.New(fmt.Sprintf("Error writing audio frame at position %s: [%s]. Last audio position: %s; Last video position: %s", w.audioTimePosition, err, w.lastAudioTimePosition, w.lastVideoTimePosition))
		}
		w.lastAudioTimePosition = w.audioTimePosition
		w.audioTimePosition += f.Duration
		if !w.lastFrameAudio {
			w.lastFrameAudio = true
		}
	}
	return nil
}

func (w *MKVWriter) createMKVFile() (*matroska.Matroska, error) {
	t := time.Now()
	path := w.dstDir + "/" + t.Format("2006/01/02/15-04-05.") + w.FileSuff + ".mkv"
	directoryPath := filepath.Dir(path)
	err := os.MkdirAll(directoryPath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("Error creating directory: %v\n", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot create file %v: %v\n", path, err)
	}
	var vt matroska.Track
	if w.IsHEVC {
		vt = matroska.NewTrackH265()
	} else {
		vt = matroska.NewTrackH264()
	}
	tracks := []matroska.Track{vt}
	if w.HasAudio {
		tracks = append(tracks, matroska.NewTrackPCMA(int(dvrframe.ExpectedAudioSampleRate), 1))
	}
	return matroska.Open(file, tracks...)
}
