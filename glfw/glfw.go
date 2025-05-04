package glfw

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/core/std"
	"github.com/ignite-laboratories/hydra"
	"log"
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

var Windows map[uint64]*Head
var Synchro std.Synchro

var GLVersion struct {
	Major int
	Minor int
	Core  bool
}

var once sync.Once
var mutex sync.Mutex
var running bool

func reset() {
	once = sync.Once{}
	Windows = make(map[uint64]*Head)
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

		core.Verbosef(ModuleName, "sparking GLFW integration\n")
		running = true

		if err := glfw.Init(); err != nil {
			log.Fatalf("Failed to initialize GLFW: %v", err)
		}
		defer glfw.Terminate()

		glfw.WindowHint(glfw.ContextCreationAPI, glfw.EGLContextAPI)
		glfw.WindowHint(glfw.ContextVersionMajor, GLVersion.Major)
		glfw.WindowHint(glfw.ContextVersionMinor, GLVersion.Minor)
		if GLVersion.Core {
			glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		} else {
			glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
			glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLAnyProfile)
		}

		wg.Done()

		for core.Alive && running {
			Synchro.Engage() // Listen for external execution

			glfw.PollEvents()
		}

		core.Verbosef(ModuleName, "GLFW integration stopped\n")
		reset() // Reset for re-activation
	}()
	wg.Wait()
}

func (h *Head) setImpulsable(impulsable core.Impulsable) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	if h.Impulsable == nil {
		core.Verbosef(ModuleName, "[%d] provided an impulsable\n", h.ID)
		h.Impulsable = impulsable
		go h.start()
	}
}

func CreateWindow(engine *core.Engine, title string, size *std.XY[int], pos *std.XY[int], potential core.Potential, muted bool) *Head {
	Activate()

	// TODO: Position - In GLFW, this requires setting the window hint to hidden, then setting the position, then "showing" the window at that location
	// Yes, GLFW is really sluggish compared to SDL - don't stress too much on this part right now

	var handle *glfw.Window
	Synchro.Send(func() {
		mutex.Lock()
		defer mutex.Unlock()

		if size == nil {
			size = &hydra.DefaultSize
		}

		h, err := glfw.CreateWindow(size.X, size.Y, title, nil, nil)
		if err != nil {
			core.Fatalf(ModuleName, "failed to create GLFW window: %v\n", err)
		}
		handle = h
	})

	h := &Head{}
	h.SetImpulsable = h.setImpulsable
	h.Synchro = make(std.Synchro)
	h.Definition.Handle = handle
	h.System = core.CreateSystem(engine, func(ctx core.Context) {
		if core.Alive && h.Alive {
			h.Synchro.Send(func() {
				if h.Impulsable != nil {
					h.Impulsable.Impulse(ctx)
					h.Definition.Handle.SwapBuffers()
				}
			})
		}
	}, potential, muted)
	Windows[h.ID] = h

	core.Verbosef(ModuleName, "window [%d] created\n", h.ID)
	return h
}

func CreateFullscreenWindow(engine *core.Engine, title string, potential core.Potential, muted bool) *Head {
	Activate()

	var handle *glfw.Window
	Synchro.Send(func() {
		mutex.Lock()
		defer mutex.Unlock()

		h, err := glfw.CreateWindow(hydra.DefaultSize.X, hydra.DefaultSize.Y, title, nil, nil)
		if err != nil {
			core.Fatalf(ModuleName, "failed to create GLFW window: %v\n", err)
		}
		handle = h
	})

	h := &Head{}
	h.SetImpulsable = h.setImpulsable
	h.Synchro = make(std.Synchro)
	h.Definition.Handle = handle
	h.System = core.CreateSystem(engine, func(ctx core.Context) {
		if core.Alive && h.Alive {
			h.Synchro.Send(func() {
				if h.Impulsable != nil {
					h.Impulsable.Impulse(ctx)
					h.Definition.Handle.SwapBuffers()
				}
			})
		}
	}, potential, muted)
	Windows[h.ID] = h

	core.Verbosef(ModuleName, "fullscreen window [%d] created\n", h.ID)
	return h
}

func (h *Head) start() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	h.Definition.Handle.MakeContextCurrent()

	//sdl.GLSetSwapInterval(1)

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		core.Fatalf(ModuleName, "failed to initialize OpenGL: %v\n", err)
	}

	glVersion := gl.GoStr(gl.GetString(gl.VERSION))
	core.Verbosef(ModuleName, "[%d] initialized with %s\n", h.ID, glVersion)

	h.Impulsable.Initialize()
	for core.Alive && h.Alive && !h.Definition.Handle.ShouldClose() {
		h.Impulsable.Lock()
		h.Synchro.Engage()
		h.Impulsable.Unlock()

		// GL threads don't need to operate at more than 1kHz
		// Why waste the cycles?
		time.Sleep(time.Millisecond)
	}
	h.Impulsable.Cleanup()
	delete(Windows, h.ID)
	h.destroy()
	core.Verbosef(ModuleName, "window [%d] cleaned up\n", h.ID)
}
