// Package glfw provides a way to create impulsable openGL contexts using GLFW
package glfw

import (
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/hydra"
)

var ModuleName = "glfw"

func init() {
	hydra.Report()
	core.SubmoduleReport(hydra.ModuleName, ModuleName)
}

func Report() {}
