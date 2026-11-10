package giode

import (
	"errors"
	"image"
	"unsafe"
)

// GPUAPI is the graphics API of a window's GPU context.
type GPUAPI int

const (
	GPUOpenGL GPUAPI = iota + 1
	GPUMetal
	GPUDirect3D11
	GPUVulkan
)

// GPUTarget is the frame Config.Underlay draws into: the window's
// render target, with its GPU context current.
type GPUTarget struct {
	API GPUAPI
	// Framebuffer is the OpenGL (ES) framebuffer object to draw into; 0
	// is the window's default framebuffer.
	Framebuffer uint32
	// Texture is the Metal MTLTexture (id<MTLTexture>) or the Direct3D
	// 11 ID3D11RenderTargetView to draw into, or the Vulkan VkImage.
	Texture uintptr
	// Size is the target's size in pixels.
	Size image.Point
	// GetProcAddress looks up an OpenGL (ES) function of the window's
	// context by name - what engines ask for to load GL themselves
	// (libmpv's get_proc_address): through ANGLE's libEGL.dll on
	// Windows, EGL on Linux and Android, the OpenGL framework on macOS
	// and OpenGLES on iOS. It returns nil for a name it doesn't know.
	// It's nil for other APIs.
	GetProcAddress func(name string) unsafe.Pointer
}

// gpuTarget makes the GPUTarget of a frame.
func gpuTarget(api int, fb uint32, handle uintptr, w, h int) GPUTarget {
	t := GPUTarget{API: GPUAPI(api), Framebuffer: fb, Texture: handle, Size: image.Pt(w, h)}
	if t.API == GPUOpenGL {
		t.GetProcAddress = glGetProcAddress
	}
	return t
}

// Renderer selects the GPU API of the app's window.
type Renderer int

const (
	// RendererDefault is Gio's choice: Metal on macOS and iOS, Direct3D
	// 11 on Windows, OpenGL ES elsewhere.
	RendererDefault Renderer = iota
	// RendererOpenGL asks for OpenGL (ES) on every platform, for
	// engines that render with OpenGL (libmpv, most video, 3D and map
	// engines): on Windows through ANGLE (libEGL.dll and libGLESv2.dll
	// next to the executable), on macOS and iOS through the nometal
	// build tag (renderer = "opengl" in giode.toml).
	RendererOpenGL
)

// errUnderlayUnpatched is what Run returns for Config.Underlay or
// RendererOpenGL in an app not built against giode's patched Gio.
var errUnderlayUnpatched = errors.New("giode: Config.Underlay and Config.Renderer need giode's Gio patches: build and run the app with giode build and giode run (which build against them with the giode_gio tag)")

// setupRenderer installs the underlay and renderer choice before the
// window is created.
func setupRenderer(cfg Config) error {
	if cfg.Underlay == nil && cfg.Renderer == RendererDefault {
		return nil
	}
	if cfg.Renderer == RendererOpenGL && metalOnly {
		return errors.New("giode: Config.Renderer OpenGL on macOS and iOS needs the nometal build tag: set renderer = \"opengl\" in giode.toml")
	}
	return installUnderlay(cfg.Underlay, cfg.UnderlayRelease, cfg.Renderer == RendererOpenGL)
}

// underlayFrameDone runs on the render thread after the UI is drawn
// over the underlay: the MCP screenshot reads the frame back here,
// since an offscreen render of the UI alone lacks the underlay.
func underlayFrameDone(t GPUTarget) {
	if a := currentApp.Load(); a != nil && a.mcp != nil {
		a.mcp.captureUnderlay(t)
	}
}

// Invalidate asks for a new frame. It's safe to call from any
// goroutine: an underlay's new video frame is ready, say.
func (a *App) Invalidate() {
	a.invalidate()
}
