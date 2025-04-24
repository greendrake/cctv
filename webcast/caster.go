package webcast

import (
	// "time"
	"github.com/greendrake/cctv/frame"
	"github.com/greendrake/server_client_hierarchy"
)

type CasterGetter func(cam string, ssId string) *Caster

// const (
//     NoVideoTimeOut time.Duration = 10 * time.Second
// )

type Caster struct {
	server_client_hierarchy.Node
	CamName string
}

func NewCaster() *Caster {
	caster := &Caster{}
	caster.SetIChunkHandler(caster.frameHandler)
	return caster
}

func (c *Caster) frameHandler(chunk any) {
	f := chunk.(*frame.Frame)
	if f.IsVideo {
		c.Output(f)
	}
}
