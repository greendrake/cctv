package camera

import (
	"context"
	"fmt"
	"github.com/greendrake/cctv/util"
	"github.com/greendrake/server_client_hierarchy"
	"log"
	"net"
	"slices"
	"strconv"
	"time"
)

type CamName string
type CamType string
type StreamID uint8

const (
	StreamMain  StreamID = 0
	StreamExtra StreamID = 1
)

type StreamConfig struct {
	ID      StreamID
	UseRTSP bool
}

// This struct is read into from JSON by jsonconfig.
// It is also client to CCTV (the apex Node).
type Camera struct {
	// YAML fields start
	Name     CamName        `yaml:"Name"`
	Address  string         `yaml:"Address"`
	User     string         `yaml:"User"`
	Password string         `yaml:"Password"`
	Type     CamType        `yaml:"Type"` // DVR assumed by default. Can be BITVISION
	UseRTSP  bool           `yaml:"UseRTSP"`
	HasAudio bool           `yaml:"HasAudio"`
	Streams  []StreamConfig `yaml:"Streams"`
	Save     []StreamID     `yaml:"Save"`    // Streams to save to files
	WebCast  []StreamID     `yaml:"WebCast"` // Streams to broadcast via MSE
	// YAML fields end

	server_client_hierarchy.Node
	dstDir     string
	IsDisabled bool
}

func isReachable(ctx context.Context, ip string, port int) bool {
	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var dialer net.Dialer
	conn, err := dialer.DialContext(dialCtx, "tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func StreamID2String(sId StreamID) string {
	return strconv.Itoa(int(sId))
}

func (c *Camera) HasAnythingToDo() bool {
	return !c.IsDisabled && (len(c.Save) > 0 || len(c.WebCast) > 0)
}

func (c *Camera) Init(baseDir string) {
	// Even though Camera acts as a server, we don't want it to stop when all clients removed.
	// It will be started automatically when added to CCTV.
	c.SetPrincipallyClient(true)
	c.dstDir = baseDir + "/" + string(c.Name)
	c.GetNode().ID = "Camera [" + string(c.Name) + "]"
	if c.User == "" {
		c.User = "admin"
	}
	c.SetTask(func(ch chan bool) {
		for {
			select {
			case <-ch:
				return
			case <-c.Node.Ctx.Done():
				<-ch
				return
			default:
				if len(c.Save) > 0 && !c.isSavingAllThatItShould() {
					if c.isOnline() {
						for _, s := range c.Save {
							c.GetStream(s)
						}
					} else {
						log.Printf("%v is offline, retrying in 5s...", c.GetNode().ID)
						util.SleepCtx(c.Node.Ctx, 5000*time.Millisecond)
					}
				} else {
					// Nothing needs to be done. Stand by.
					util.SleepCtx(c.Node.Ctx, 100*time.Millisecond)
				}
			}
		}
	})
}

func (c *Camera) GetStream(sId StreamID) *Stream {
	for _, _stream := range c.Clients {
		stream := _stream.(*Stream)
		if stream.ID == sId && !stream.IsStopping() {
			return stream
		}
	}
	// Stream does not exist yet.
	UseRTSP := false
	for _, s := range c.Streams {
		if s.ID == sId && s.UseRTSP {
			UseRTSP = true
			break
		}
	}
	stream := &Stream{
		ID:      sId,
		UseRTSP: UseRTSP,
		camera:  c,
	}
	stream.Init()
	c.AddClient(stream)
	return stream
}

func (c *Camera) getPingArgs() (string, int) {
	var port int
	if c.Type == "BITVISION" || c.UseRTSP {
		port = 554
	} else {
		port = 34567
	}
	return c.Address, port
}

func (c *Camera) isOnline() bool {
	a, b := c.getPingArgs()
	return isReachable(c.Node.Ctx, a, b)
}

func (c *Camera) isSavingAllThatItShould() bool {
	shouldBeSaving := make([]StreamID, len(c.Save))
	copy(shouldBeSaving, c.Save)
	for _, _s := range c.Clients {
		stream := _s.(*Stream)
		i := slices.Index(shouldBeSaving, stream.ID)
		if i > -1 {
			shouldBeSaving = slices.Delete(shouldBeSaving, i, 1)
		}
	}
	return len(shouldBeSaving) == 0
}

// Depending on configuration, this task can be comprised of up to 4 subtasks:
// 1. Save the main stream to MKV.
// 2. Save the extra stream to MKV.
// 3. Cast the main stream to MSE (if there are any WS clients)
// 4. Cast the extra stream to MSE (if there are any WS clients)
// At any time, there will be no more than 1 DVRIP (or RTSP) monitor (per stream),
// which can be consumed by either or both MKV saver or MSE streamer.
