package glfw

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/ignite-laboratories/core"
	"github.com/ignite-laboratories/core/std"
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

var DefaultSize = std.XY[int]{
	X: 640,
	Y: 480,
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

func CreateWindow(engine *core.Engine, title string, size *std.XY[int], pos *std.XY[int], impulsable core.Impulsable, potential core.Potential, muted bool) *Head {
	Activate()

	var handle *glfw.Window
	Synchro.Send(func() {
		mutex.Lock()
		defer mutex.Unlock()

		h, err := glfw.CreateWindow(size.X, size.Y, title, nil, nil)
		if err != nil {
			core.Fatalf(ModuleName, "failed to create GLFW window: %v\n", err)
		}
		handle = h
	})

	head := &Head{}
	head.Synchro = make(std.Synchro)
	head.Definition = handle
	head.System = core.CreateSystem(engine, func(ctx core.Context) {
		if head.Alive {
			head.Synchro.Send(func() {
				impulsable.Impulse(ctx)
				head.Definition.SwapBuffers()
			})
		}
	}, potential, muted)
	Windows[head.ID] = head

	core.Verbosef(ModuleName, "window [%d] created\n", head.ID)
	go head.start(impulsable)

	return head
}

func CreateFullscreenWindow(engine *core.Engine, title string, impulsable core.Impulsable, potential core.Potential, muted bool) *Head {
	Activate()

	var handle *glfw.Window
	Synchro.Send(func() {
		mutex.Lock()
		defer mutex.Unlock()

		h, err := glfw.CreateWindow(DefaultSize.X, DefaultSize.Y, title, nil, nil)
		if err != nil {
			core.Fatalf(ModuleName, "failed to create GLFW window: %v\n", err)
		}
		handle = h
	})

	head := &Head{}
	head.Synchro = make(std.Synchro)
	head.Definition = handle
	head.System = core.CreateSystem(engine, func(ctx core.Context) {
		if head.Alive {
			head.Synchro.Send(func() {
				impulsable.Impulse(ctx)
				head.Definition.SwapBuffers()
			})
		}
	}, potential, muted)
	Windows[head.ID] = head

	core.Verbosef(ModuleName, "fullscreen window [%d] created\n", head.ID)
	go head.start(impulsable)

	return head
}

func (w *Head) start(impulsable core.Impulsable) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	w.Definition.MakeContextCurrent()

	//sdl.GLSetSwapInterval(1)

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		core.Fatalf(ModuleName, "failed to initialize OpenGL: %v\n", err)
	}

	glVersion := gl.GoStr(gl.GetString(gl.VERSION))

	core.Verbosef(ModuleName, "[%d] initialized with %s\n", w.ID, glVersion)
	impulsable.Initialize()
	for core.Alive && w.Alive && !w.Definition.ShouldClose() {
		w.Synchro.Engage()

		// GL threads don't need to operate more than 1kHz
		// Why waste the cycles?
		//time.Sleep(time.Millisecond)
	}
	impulsable.Cleanup()
	delete(Windows, w.ID)
	w.destroy()
	core.Verbosef(ModuleName, "window [%d] cleaned up\n", w.ID)
}
