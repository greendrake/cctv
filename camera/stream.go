package camera

import (
	"errors"
	"fmt"
	"github.com/greendrake/cctv/dvr"
	"github.com/greendrake/cctv/rtsp"
	"github.com/greendrake/cctv/util"
	"github.com/greendrake/cctv/webcast"
	"github.com/greendrake/server_client_hierarchy"
	"log"
	"slices"
	"sync"
	"time"
)

// Stream represents either the main or the extra stream/channel of the camera.
// Clients/subscribers can be attached to it e.g. MKV writers and MSE re-translators.

// This struct is client to Camera

const (
	noVideoTimeOut time.Duration = 500 * time.Millisecond
)

type Stream struct {
	server_client_hierarchy.Node
	ID      StreamID
	camera  *Camera
	UseRTSP bool
	// Caster puppet-masters webcast clients. It exists only if there is at least one client.
	caster *webcast.Caster
	// MKVWriter writes video to MKV files.
	mkvWriter *MKVWriter
	// Monitor pulls video from the camera. It exists at all times the stream node is running.
	// And the stream node is running/exists only if there are webcast clients (if webcast is configured at all)
	// or if MKV writing ("Save") is configured.
	monitor          Monitor
	monitorMakeMutex sync.Mutex
	casterMakeMutex  sync.Mutex
	// noVideoTimer *time.Timer
	// fc int
}

func (s *Stream) makeMonitor() {
	s.monitorMakeMutex.Lock()
	defer s.monitorMakeMutex.Unlock()
	if s.monitor == nil {
		success := false
		for success == false && !s.camera.IsDisabled {
			success = s.tryToMakeMonitor()
			if !success && !util.SleepCtx(s.Node.Ctx, 300*time.Millisecond) {
				break
			}
		}
	}
}

func (s *Stream) tryToMakeMonitor() bool {
	var err error
	if s.camera.Type == "BITVISION" || s.camera.UseRTSP || s.UseRTSP {
		s.monitor, err = rtsp.NewMonitor(s.Node.Ctx, s.getRTSPURI())
	} else {
		s.monitor, err = dvrip.NewMonitor(s.Node.Ctx, s.camera.Address, StreamID2String(s.ID), s.camera.User, s.camera.Password)
	}
	if err == nil {
		return true
	} else {
		s.monitor = nil
		if _, ok := err.(*util.WrongCredentialsError); ok {
			log.Printf("Wrong credentials for camera %v", s.camera.Name)
			s.camera.IsDisabled = true
			go s.camera.Stop()
		}
		return false
	}
}

func (s *Stream) stopMonitor(reason error) {
	if s.monitor != nil {
		m := s.monitor
		s.monitor = nil
		m.ShutDown()
	}
}

func (s *Stream) Init() {
	s.GetNode().ID = fmt.Sprintf("Stream [%v]:%v", s.camera.Name, s.ID)
	s.SetTask(func(ch chan bool) {
		defer s.stopMonitor(errors.New("stream task finished"))
		s.makeMonitor()
		if s.monitor != nil && slices.Contains(s.camera.Save, s.ID) {
			s.mkvWriter = &MKVWriter{
				dstDir:   s.camera.dstDir,
				HasAudio: s.camera.HasAudio,
				FileSuff: StreamID2String(s.ID),
			}
			s.mkvWriter.Init()
			s.AddClient(s.mkvWriter)
		}
		for {
			select {
			case <-ch:
				return
			case <-s.Node.Ctx.Done():
				<-ch
				return
			default:
				if s.monitor != nil {
					f, err := s.monitor.GetFrame()
					if err == nil {
						s.Output(f)
					} else {
						s.stopMonitor(err)
						go s.Stop()
						<-ch
						return
					}
				}
			}
		}
	})
}

func (s *Stream) getRTSPURI() string {
	// rtsp://192.168.72.150/user=admin&password=&channel=1&stream=0.sdp
	// rtsp://rtsp:rtsp1234@192.168.72.132:554/0
	uri := "rtsp://"
	streamID := StreamID2String(s.ID)
	if s.camera.Type == "BITVISION" {
		uri += s.camera.User + ":" + s.camera.Password + "@"
	}
	uri = uri + s.camera.Address
	if s.camera.Type == "BITVISION" {
		uri += ":554/" + streamID
	} else {
		uri += "/user=" + s.camera.User + "&password=" + s.camera.Password + "&channel=1&stream=" + streamID + ".sdp"
	}
	return uri
}

func (s *Stream) GetCaster() *webcast.Caster {
	s.casterMakeMutex.Lock()
	defer s.casterMakeMutex.Unlock()
	if s.caster == nil {
		s.makeMonitor()
		// If there was a mutex lock and the app was interrupted, s.monitor will still be nil here, so:
		if s.monitor != nil {
			cId := "Caster [" + string(s.camera.Name) + ":" + StreamID2String(s.ID) + "]"
			s.caster = webcast.NewCaster()
			s.caster.CamName = string(s.camera.Name)
			s.caster.GetNode().ID = cId
			s.caster.On("stop", func(args ...any) {
				s.caster = nil
			})
			s.AddClient(s.caster)
		}
	}
	return s.caster
}
