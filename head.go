package hydra

import (
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/core/std"
	"sync"
)

type Head[TDefinition any] struct {
	*core.System

	Definition    TDefinition
	Synchro       std.Synchro
	Impulsable    core.Impulsable
	SetImpulsable func(core.Impulsable)
	Mutex         sync.Mutex
}
