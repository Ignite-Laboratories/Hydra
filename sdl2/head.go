package sdl2

import (
	"github.com/ignite-laboratories/hydra"
	"github.com/veandco/go-sdl2/sdl"
)

type Head hydra.Head[*sdl.Window, sdl.GLContext, sdl.Event]
