package hydra

import "github.com/ignite-laboratories/core/std"

// DefaultSize sets the default window size for new windows.
//
// If not overridden, it defaults to 640x480px
var DefaultSize = std.XY[int]{
	X: 640,
	Y: 480,
}
