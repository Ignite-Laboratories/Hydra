package hydra

import (
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/core/std"
)

type Head[THandle any, TContext any, TEvent any] struct {
	*core.System

	Handle  THandle
	Context TContext
	Synchro std.Synchro

	EventHandler func(TEvent)
}
