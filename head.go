package hydra

import (
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/core/std"
)

type Head[TDefinition any] struct {
	*core.System

	Definition TDefinition
	Synchro    std.Synchro
}
