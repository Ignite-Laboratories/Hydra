package sdl2

import "github.com/veandco/go-sdl2/sdl"

type SDLDefinition struct {
	Handle       *sdl.Window
	Context      sdl.GLContext
	EventHandler func(event sdl.Event)
}
