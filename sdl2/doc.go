// Package sdl2 provides a way to create impulsable openGL contexts using SDL2
package sdl2

import (
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/hydra"
)

var ModuleName = "sdl2"

func init() {
	hydra.Report()
	core.SubmoduleReport(hydra.ModuleName, ModuleName)
}

func Report() {}
