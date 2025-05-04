package glfw

import (
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/core/std"
	"github.com/ignite-laboratories/hydra"
	"sync"
)

type Head struct {
	hydra.Head[GLFWDefinition]
	mutex sync.Mutex
}

func (h *Head) destroy() {
	Synchro.Send(func() {
		h.Definition.Handle.Destroy()
	})
}

func Create(engine *core.Engine, fullscreen bool, framePotential core.Potential, title string, size *std.XY[int], pos *std.XY[int]) *hydra.Head[GLFWDefinition] {
	if fullscreen {
		return &CreateFullscreenWindow(engine, title, framePotential, false).Head
	}
	return &CreateWindow(engine, title, size, pos, framePotential, false).Head
}
