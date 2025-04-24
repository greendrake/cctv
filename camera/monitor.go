package camera

import (
	"github.com/greendrake/cctv/frame"
)

type Monitor interface {
	GetFrame() (*frame.Frame, error)
	ShutDown()
}
