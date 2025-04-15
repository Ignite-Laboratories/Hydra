// Package hydra provides tools for creating multithreaded architectures in the JanOS ecosystem.
package hydra

import (
	"github.com/ignite-laboratories/core"
)

var ModuleName = "hydra"

func init() {
	core.ModuleReport(ModuleName)
}

func Report() {}
