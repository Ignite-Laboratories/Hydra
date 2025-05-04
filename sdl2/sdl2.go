package sdl2

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/core/std"
	"github.com/ignite-laboratories/hydra"
	"github.com/veandco/go-sdl2/sdl"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

func init() {
	GLVersion.Major = 3
	GLVersion.Minor = 1
	GLVersion.Core = false
	reset()
}

var Windows map[uint64]*Head
var Synchro std.Synchro

var GLVersion struct {
	Major int
	Minor int
	Core  bool
}

var once sync.Once
var running bool

func (h *Head) destroy() {
	// Queue up the destruction action on the SDL thread...
	go Synchro.Send(func() {
		h.Definition.Handle.Destroy()
	})

	// ...then send an artificial event to trigger the event poll to cycle
	userEvent := sdl.UserEvent{
		Type:  destructionEvent,
		Code:  int32(core.NextID()),
		Data1: unsafe.Pointer(uintptr(h.ID)),
		Data2: unsafe.Pointer(uintptr(h.WindowID)),
	}
	sdl.PushEvent(&userEvent)
}

func reset() {
	once = sync.Once{}
	Windows = make(map[uint64]*Head)
	Synchro = make(std.Synchro)
	running = false
}

// HasNoWindows provides a potential that returns true when all the windows have been globally closed.
func HasNoWindows(ctx core.Context) bool {
	if len(Windows) == 0 {
		running = false
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

var destructionEvent uint32

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

		destructionEvent = sdl.RegisterEvents(1)

		wg.Done()

		for core.Alive && running {
			Synchro.Engage() // Listen for external execution

			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				if !core.Alive || !running {
					break
				}

				// Pass all events along to the window event handlers
				for _, sys := range Windows {
					if sys.Definition.EventHandler != nil {
						sys.Definition.EventHandler(event)
					}
				}

				// Handle the close events directly
				switch e := event.(type) {
				case *sdl.WindowEvent:
					for _, sys := range Windows {
						if sys.WindowID == e.WindowID {
							if e.Event == sdl.WINDOWEVENT_CLOSE {
								sys.Stop()
								go func() {
									// Force the window's main loop to cycle
									time.Sleep(time.Millisecond)
									sys.Synchro.Send(func() {})
								}()
								break
							}
						}
					}
				case *sdl.UserEvent:
					if e.Type == destructionEvent {
						// a window was fully destroyed
						id := uint64(uintptr(e.Data1))
						windowID := uint64(uintptr(e.Data2))
						delete(Windows, id)
						core.Verbosef(ModuleName, "window [%d.%d] cleaned up\n", windowID, id)
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

func CreateWindow(engine *core.Engine, title string, size *std.XY[int], pos *std.XY[int], potential core.Potential, muted bool) *Head {
	Activate()

	var handle *sdl.Window
	Synchro.Send(func() {
		var posX = sdl.WINDOWPOS_UNDEFINED
		var posY = sdl.WINDOWPOS_UNDEFINED
		if pos != nil {
			posX = pos.X
			posY = pos.Y
		}

		var sizeX = hydra.DefaultSize.X
		var sizeY = hydra.DefaultSize.Y
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

	h := &Head{}
	h.SetImpulsable = h.setImpulsable
	h.Synchro = make(std.Synchro)
	h.Definition.Handle = handle
	h.WindowID, _ = handle.GetID()
	h.System = core.CreateSystem(engine, func(ctx core.Context) {
		if core.Alive && h.Alive {
			h.Synchro.Send(func() {
				if h.Impulsable != nil {
					h.Impulsable.Impulse(ctx)
					h.Definition.Handle.GLSwap()
				}
			})
		}
	}, potential, muted)
	Windows[h.ID] = h

	core.Verbosef(ModuleName, "window [%d.%d] created\n", h.WindowID, h.ID)
	return h
}

func (h *Head) setImpulsable(impulsable core.Impulsable) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	if h.Impulsable == nil {
		core.Verbosef(ModuleName, "[%d.%d] provided an impulsable\n", h.WindowID, h.ID)
		h.Impulsable = impulsable
		go h.start()
	}
}

func CreateFullscreenWindow(engine *core.Engine, title string, potential core.Potential, muted bool) *Head {
	Activate()

	var handle *sdl.Window
	Synchro.Send(func() {
		h, err := sdl.CreateWindow(
			title,
			sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
			int32(hydra.DefaultSize.X), int32(hydra.DefaultSize.Y),
			sdl.WINDOW_OPENGL|sdl.WINDOW_FULLSCREEN_DESKTOP,
		)
		if err != nil {
			core.Fatalf(ModuleName, "failed to create SDL window: %v\n", err)
		}
		handle = h
	})

	h := &Head{}
	h.SetImpulsable = h.setImpulsable
	h.Synchro = make(std.Synchro)
	h.Definition.Handle = handle
	h.WindowID, _ = handle.GetID()
	h.System = core.CreateSystem(engine, func(ctx core.Context) {
		if core.Alive && h.Alive {
			h.Synchro.Send(func() {
				if h.Impulsable != nil {
					h.Impulsable.Impulse(ctx)
					h.Definition.Handle.GLSwap()
				}
			})
		}
	}, potential, muted)
	Windows[h.ID] = h

	core.Verbosef(ModuleName, "fullscreen window [%d.%d] created\n", h.WindowID, h.ID)
	return h
}

func (h *Head) start() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	glContext, err := h.Definition.Handle.GLCreateContext()
	if err != nil {
		core.Fatalf(ModuleName, "failed to create OpenGL context: %v\n", err)
	}

	sdl.GLSetSwapInterval(1)

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		core.Fatalf(ModuleName, "failed to initialize OpenGL: %v\n", err)
	}

	h.Definition.Context = glContext
	defer sdl.GLDeleteContext(glContext)

	glVersion := gl.GoStr(gl.GetString(gl.VERSION))
	core.Verbosef(ModuleName, "[%d.%d] initialized with %s\n", h.WindowID, h.ID, glVersion)

	h.Impulsable.Initialize()
	for core.Alive && h.Alive {
		h.Impulsable.Lock()
		h.Synchro.EngageOnce()
		h.Impulsable.Unlock()
	}
	h.Impulsable.Cleanup()
	h.destroy()
}
