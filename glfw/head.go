package glfw

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/ignite-laboratories/hydra"
)

type Head hydra.Head[*glfw.Window]

func (w *Head) destroy() {
	Synchro.Send(func() {
		w.Definition.Destroy()
	})
}
