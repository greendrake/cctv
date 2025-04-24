package webcast

import (
	// "log"
	// "fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/greendrake/cctv/frame"
	"github.com/greendrake/cctv/muxer/h265"
	"github.com/greendrake/cctv/muxer/mp4"
	"github.com/greendrake/server_client_hierarchy"
	"golang.org/x/net/websocket"
	"sync"
	"time"
)

const (
	ClockRate uint32 = 90000
	KeyByte   byte   = byte(255)
)

// Client is a principally client Node.
// It runs standalone initially, and, upon establishing connection with the browser, it attaches as a client to the Caster.
type Client struct {
	server_client_hierarchy.Node
	caster             *Caster
	wsReadyChannel     chan bool
	stopCommandChannel chan bool
	ws                 *websocket.Conn
	wsReady            bool
	muxer              *mp4.Muxer
	trackID            byte
	started            bool
	wsWriteMutex       sync.Mutex
}

// Each client needs its own muxer because re-connecting clients need to start receiving
// what a muxer produces from the beginning, not after a while.
// Probably because of packet ordinal sequence numbers or something, dunno exactly.
// Otherwise muxer could sit in the Caster, and each Client would just pass []byte payload to the browsers.

func NewClient(c *gin.Context, caster *Caster) *Client {
	client := &Client{
		caster:         caster,
		wsReadyChannel: make(chan bool),
		trackID:        byte(0),
	}
	client.GetNode().ID = "Client " + uuid.New().String() + ", caster " + caster.GetNode().ID
	client.SetPrincipallyClient(true)
	client.SetTask(func(ch chan bool) {
		client.stopCommandChannel = ch
		handler := websocket.Handler(client.wsHandler)
		handler.ServeHTTP(c.Writer, c.Request)
	})
	client.SetIChunkHandler(client.videoChunkHandler)
	caster.AddClient(client) // client will start receiving frames from the caster now. They will build up in the queue until the muxer is populated
	// log.Printf("Creating client %v", client.GetNode().ID)
	// client.On("stop", func(args ...any) {
	//     log.Printf("Stopped client %v", client.GetNode().ID)
	// })
	return client
}

func (c *Client) videoChunkHandler(chunk any) {
	if !c.wsReady {
		<-c.wsReadyChannel
		c.wsReady = true
	}
	f := chunk.(*frame.Frame)
	payload := h265.EncodeToAVCC(*f.Data)
	if f.IsVideoKeyFrame {
		if !c.started {
			c.muxer = &mp4.Muxer{}
			c.wsWriteMutex.Lock()
			c.started = true
			c.writeToWS(c.muxer.GetInit(payload, ClockRate))
		}
	}
	if c.started {
		c.wsWriteMutex.Lock()
		// log.Printf("Frame duration %v", uint32(f.Duration))
		// Normally the duration needs to be divided by 10,000, but we intentionally feed a shorter
		// duration (by dividing by a larger number e.g. 12,000) so that the client does not lag behind
		// but always feels hungry for new frames, and does display them as soon as they arrive:
		c.writeToWS(c.muxer.GetPayload(c.trackID, &payload, uint32(f.Duration)/12000))
	}
}

func (c *Client) writeToWS(data []byte) {
	defer c.wsWriteMutex.Unlock()
	if c.ws != nil {
		err := c.ws.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
		if err != nil {
			// This will loop back to videoChunkHandler() in order to flush the input queue,
			// so send it to its own goroutine to avoid deadlock:
			// log.Printf("WS write deadline error, stopping %v", c.GetNode().ID)
			go c.stopAndClose()
			return
		}
		err = websocket.Message.Send(c.ws, data)
		if err != nil {
			// log.Printf("Message send error, stopping %v", c.GetNode().ID)
			go c.stopAndClose()
		}
	}
}

func (c *Client) wsHandler(ws *websocket.Conn) {
	defer c.stopAndClose()
	c.ws = ws
	// This is needed to detect WS disconnection by the browser,
	// and also to receive reset commands:
	go func() {
		var message string
		var err error
		for {
			err = websocket.Message.Receive(ws, &message)
			if message == "reset" {
				// log.Printf("Reset request %v", c.GetNode().ID)
				c.wsWriteMutex.Lock()
				c.started = false
				// Send cut-off byte so that the client knows when to expect the next muxer init payload
				c.writeToWS([]byte{0xFF})
			}
			if err != nil {
				// log.Printf("Message receive error %v", c.GetNode().ID)
				c.stopAndClose()
				return
			}
		}
	}()
	select {
	case <-c.Node.Ctx.Done():
		// log.Printf("Main context done, waiting for stopCommandChannel; %v", c.GetNode().ID)
		<-c.stopCommandChannel
	case c.wsReadyChannel <- true: // Wait for the chunk handler to receive from this channel. Note that it may never do so if there are no chunks!
		// log.Printf("Sent to wsReadyChannel; %v", c.GetNode().ID)
		<-c.stopCommandChannel
	case <-c.stopCommandChannel:
		// log.Printf("StopCommandChannel received; %v", c.GetNode().ID)
	}
}

func (c *Client) stopAndClose() {
	if c.ws != nil {
		defer c.wsWriteMutex.Unlock()
		c.wsWriteMutex.Lock()
		_ws := c.ws
		c.ws = nil
		_ws.Close()
		_ws = nil
	}
	c.Stop()
}
