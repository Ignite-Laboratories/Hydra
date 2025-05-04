package sdl2

import (
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/core/std"
	"github.com/ignite-laboratories/hydra"
	"sync"
)

type Head struct {
	hydra.Head[SDLDefinition]
	mutex    sync.Mutex
	WindowID uint32
}

func Create(engine *core.Engine, fullscreen bool, framePotential core.Potential, title string, size *std.XY[int], pos *std.XY[int]) *hydra.Head[SDLDefinition] {
	if fullscreen {
		return &CreateFullscreenWindow(engine, title, framePotential, false).Head
	}
	return &CreateWindow(engine, title, size, pos, framePotential, false).Head
}
