package sdl2

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/core/std"
	"github.com/veandco/go-sdl2/sdl"
	"runtime"
	"sync"
	"time"
)

func init() {
	GLVersion.Major = 3
	GLVersion.Minor = 1
	GLVersion.Core = false
	reset()
}

var Windows map[uint64]*Window
var Synchro std.Synchro

var GLVersion struct {
	Major int
	Minor int
	Core  bool
}

var DefaultSize = std.XY[int]{
	X: 640,
	Y: 480,
}

var once sync.Once
var running bool

type Window struct {
	*Head

	WindowID uint32

	EventHandler func(sdl.Event)
}

func (w *Window) destroy() {
	Synchro.Send(func() {
		w.Handle.Destroy()
	})
}

func reset() {
	once = sync.Once{}
	Windows = make(map[uint64]*Window)
	Synchro = make(std.Synchro)
	running = false
}

// HasNoWindows provides a potential that returns true when all the windows have been globally closed.
func HasNoWindows(ctx core.Context) bool {
	if len(Windows) == 0 {
		// Give SDL a second to clean up before reporting a stop condition.
		running = false
		time.Sleep(time.Second)
		return true
	}
	return false
}

func Activate() {
	once.Do(run)
}

func Stop() {
	running = false
}

func run() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		core.Verbosef(ModuleName, "sparking SDL2 integration\n")
		running = true

		// Initialize SDL
		if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
			core.Fatalf(ModuleName, "failed to initialize SDL2: %v\n", err)
		}
		//defer sdl.Quit()

		// Set OpenGL attributes
		sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, GLVersion.Major)
		sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, GLVersion.Minor)
		if GLVersion.Core {
			sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
		} else {
			sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_ES)
		}
		sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)
		sdl.GLSetAttribute(sdl.GL_DEPTH_SIZE, 24)

		driver, _ := sdl.GetCurrentVideoDriver()
		core.Verbosef(ModuleName, "SDL video driver: %s\n", driver)
		driver = sdl.GetCurrentAudioDriver()
		core.Verbosef(ModuleName, "SDL audio driver: %s\n", driver)

		wg.Done()

		for core.Alive && running {
			Synchro.Engage() // Listen for external execution

			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				if !core.Alive || !running {
					break
				}

				// Pass all events along to the window event handlers
				for _, sys := range Windows {
					if sys.EventHandler != nil {
						sys.EventHandler(event)
					}
				}

				// Handle the close events directly
				switch e := event.(type) {
				case *sdl.WindowEvent:
					for _, sys := range Windows {
						if sys.WindowID == e.WindowID {
							if e.Event == sdl.WINDOWEVENT_CLOSE {
								sys.Stop()
								break
							}
						}
					}
				case *sdl.KeyboardEvent:
					if e.Type == sdl.KEYDOWN {
						switch e.Keysym.Sym {
						case sdl.K_ESCAPE:
							for _, sys := range Windows {
								sys.Stop()
							}
						}
					}
				}
			}
		}

		core.Verbosef(ModuleName, "SDL2 integration stopped\n")
		reset() // Reset for re-activation
	}()
	wg.Wait()
}

func CreateWindow(engine *core.Engine, title string, size *std.XY[int], pos *std.XY[int], impulsable core.Impulsable, potential core.Potential, muted bool) *Head {
	Activate()

	var handle *sdl.Window
	Synchro.Send(func() {
		var posX = sdl.WINDOWPOS_UNDEFINED
		var posY = sdl.WINDOWPOS_UNDEFINED
		if pos != nil {
			posX = pos.X
			posY = pos.Y
		}

		var sizeX = DefaultSize.X
		var sizeY = DefaultSize.Y
		if size != nil {
			sizeX = size.X
			sizeY = size.Y
		}

		h, err := sdl.CreateWindow(
			title,
			int32(posX), int32(posY),
			int32(sizeX), int32(sizeY),
			sdl.WINDOW_OPENGL|sdl.WINDOW_RESIZABLE,
		)
		if err != nil {
			core.Fatalf(ModuleName, "failed to create SDL window: %v\n", err)
		}
		handle = h
	})

	w := &Window{}
	w.Head = &Head{}
	w.Synchro = make(std.Synchro)
	w.Handle = handle
	w.WindowID, _ = handle.GetID()
	w.System = core.CreateSystem(engine, func(ctx core.Context) {
		if w.Alive {
			w.Synchro.Send(func() {
				impulsable.Impulse(ctx)
				w.Handle.GLSwap()
			})
		}
	}, potential, muted)
	Windows[w.ID] = w

	core.Verbosef(ModuleName, "window [%d.%d] created\n", w.WindowID, w.ID)
	go w.start(impulsable)

	return w.Head
}

func CreateFullscreenWindow(engine *core.Engine, title string, impulsable core.Impulsable, potential core.Potential, muted bool) *Head {
	Activate()

	var handle *sdl.Window
	Synchro.Send(func() {
		h, err := sdl.CreateWindow(
			title,
			sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
			int32(DefaultSize.X), int32(DefaultSize.Y),
			sdl.WINDOW_OPENGL|sdl.WINDOW_FULLSCREEN_DESKTOP,
		)
		if err != nil {
			core.Fatalf(ModuleName, "failed to create SDL window: %v\n", err)
		}
		handle = h
	})

	w := &Window{}
	w.Head = &Head{}
	w.Synchro = make(std.Synchro)
	w.Handle = handle
	w.WindowID, _ = handle.GetID()
	w.System = core.CreateSystem(engine, func(ctx core.Context) {
		if w.Alive {
			w.Synchro.Send(func() {
				impulsable.Impulse(ctx)
				w.Handle.GLSwap()
			})
		}
	}, potential, muted)
	Windows[w.ID] = w

	core.Verbosef(ModuleName, "fullscreen window [%d.%d] created\n", w.WindowID, w.ID)
	go w.start(impulsable)

	return w.Head
}

func (w *Window) start(impulsable core.Impulsable) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	glContext, err := w.Handle.GLCreateContext()
	if err != nil {
		core.Fatalf(ModuleName, "failed to create OpenGL context: %v\n", err)
	}

	sdl.GLSetSwapInterval(1)

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		core.Fatalf(ModuleName, "failed to initialize OpenGL: %v\n", err)
	}

	glVersion := gl.GoStr(gl.GetString(gl.VERSION))

	w.Context = glContext
	defer sdl.GLDeleteContext(glContext)

	core.Verbosef(ModuleName, "[%d.%d] initialized with %s\n", w.WindowID, w.ID, glVersion)
	impulsable.Initialize()
	for core.Alive && w.Alive {
		w.Synchro.Engage()

		// GL threads don't need to operate more than 1kHz
		// Why waste the cycles?
		time.Sleep(time.Millisecond)
	}
	impulsable.Cleanup()
	delete(Windows, w.ID)
	w.destroy()
	core.Verbosef(ModuleName, "window [%d.%d] cleaned up\n", w.WindowID, w.ID)
}
