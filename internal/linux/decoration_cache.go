//go:build linux && (amd64 || arm64)

package linux

import (
	"sync"
	"time"

	"github.com/ebitengine/purego"
)

// A window GTK decorates (window.csd) paints its decorations, a blurred
// shadow and a frame, in memory with cairo, wherever it repaints. A window
// that draws with OpenGL repaints whole in every frame where its frames
// are not kept (EGL's buffer age), as in gamescope, so a surface animating
// at 60 Hz had GTK blur the shadow 60 times a second, though it never
// changed: a fifth of the main thread's time. The decorations cache keeps
// what the window painted around the area once it has settled, and paints
// that instead, the area drawing itself with OpenGL as before.

// decorationSettle is how long the window's look must stay the same before
// it is kept: longer than the transitions of the decorations' styles, as
// the shadow fading as the window loses the focus.
const decorationSettle = time.Second

var (
	decorationOnce            sync.Once
	cairoGetTarget            func(cr ptr) ptr
	cairoImageSurfaceCreate   func(format, w, h int32) ptr
	cairoCreate               func(target ptr) ptr
	cairoDestroy              func(cr ptr)
	cairoSave                 func(cr ptr)
	cairoRestore              func(cr ptr)
	cairoRectangle            func(cr ptr, x, y, w, h float64)
	cairoFill                 func(cr ptr)
	gtkWidgetGetStateFlags    func(w ptr) uint32
	gdkWindowGetState         func(w ptr) uint32
	gtkContainerPropagateDraw func(container, child, cr ptr)
	cbDecorationDraw          ptr
	cbDecorationDrawn         ptr
	cbDecorationStyle         ptr
)

// decorationKey is what the decorations look like by: the window's size
// and scale, its state (backdrop, maximized, tiled) and its style.
type decorationKey struct {
	w, h, scale int32
	area        gdkRectangle
	flags       uint32
	state       uint32
	style       uint64
}

// bands returns the rectangles of the window around the area: above,
// below, left and right of it.
func (k decorationKey) bands() [4]gdkRectangle {
	a := k.area
	return [4]gdkRectangle{
		{0, 0, k.w, a.Y},
		{0, a.Y + a.Height, k.w, k.h - a.Y - a.Height},
		{0, a.Y, a.X, a.Height},
		{a.X + a.Width, a.Y, k.w - a.X - a.Width, a.Height},
	}
}

// decorations is the cache of a surface's window.
type decorations struct {
	key   decorationKey
	since time.Time
	// bands are what the window painted around the area, in device
	// pixels: four bands, decorationKey.bands.
	bands   [4]ptr
	kept    bool
	capture bool // the paint in progress is to be kept
	style   uint64
}

const cairoFormatARGB32 = 0
const cairoOperatorSource = 1

func loadDecorationCache() {
	decorationOnce.Do(func() {
		c, t, d := libCairo, libGTK, libGDK
		mustBind(c, &cairoGetTarget, "cairo_get_target")
		mustBind(c, &cairoImageSurfaceCreate, "cairo_image_surface_create")
		mustBind(c, &cairoCreate, "cairo_create")
		mustBind(c, &cairoDestroy, "cairo_destroy")
		mustBind(c, &cairoSave, "cairo_save")
		mustBind(c, &cairoRestore, "cairo_restore")
		mustBind(c, &cairoRectangle, "cairo_rectangle")
		mustBind(c, &cairoFill, "cairo_fill")
		mustBind(t, &gtkWidgetGetStateFlags, "gtk_widget_get_state_flags")
		mustBind(d, &gdkWindowGetState, "gdk_window_get_state")
		mustBind(t, &gtkContainerPropagateDraw, "gtk_container_propagate_draw")
		b := func() *Backend { return theBackend }
		cbDecorationDraw = purego.NewCallback(func(widget, cr, data ptr) bool {
			s := b().surfaceOf(data)
			return s != nil && s.drawDecorations(widget, cr)
		})
		cbDecorationDrawn = purego.NewCallback(func(widget, cr, data ptr) bool {
			if s := b().surfaceOf(data); s != nil {
				s.keepDecorations(cr)
			}
			return false
		})
		cbDecorationStyle = purego.NewCallback(func(widget, data ptr) {
			if s := b().surfaceOf(data); s != nil {
				s.deco.style++
			}
		})
	})
}

// cacheDecorations has the window paint its decorations from the cache
// while the area draws with OpenGL: a window GTK decorates, opaque.
func (s *surface) cacheDecorations() {
	if !s.w.csd || s.w.opts.Transparent {
		return
	}
	loadDecorationCache()
	data := ptr(s.w.id)
	connect(s.w.win, "draw", cbDecorationDraw, data)
	gSignalConnectData(s.w.win, cs("draw"), cbDecorationDrawn, data, 0, 1) // G_CONNECT_AFTER
	connect(s.w.win, "style-updated", cbDecorationStyle, data)
}

// drawDecorations paints the window's decorations from the cache, and its
// child, and returns true, when the cache holds them as the window looks
// now and the window repaints whole. Otherwise it returns false, for GTK
// to paint the window, and has the paint kept once the window has looked
// the same for decorationSettle.
func (s *surface) drawDecorations(win, cr ptr) bool {
	c := &s.deco
	if !s.drawsGL() || s.inMemory || s.layers != 0 {
		c.drop()
		return false
	}
	var a gdkRectangle
	gtkWidgetGetAllocation(win, &a)
	scale := max(gtkWidgetGetScaleFactor(win), 1)
	key := decorationKey{w: a.Width, h: a.Height, scale: scale, flags: gtkWidgetGetStateFlags(win), style: c.style}
	gtkWidgetGetAllocation(s.area, &key.area)
	if gw := gtkWidgetGetWindow(win); gw != 0 {
		key.state = gdkWindowGetState(gw)
	}
	var x0, y0, x1, y1 float64
	cairoClipExtents(cr, &x0, &y0, &x1, &y1)
	whole := x0 <= 0 && y0 <= 0 && x1 >= float64(a.Width) && y1 >= float64(a.Height)
	now := time.Now()
	if key != c.key {
		c.drop()
		c.key, c.since = key, now
		return false
	}
	if !c.kept {
		// Kept from a paint of the whole window, once settled.
		c.capture = whole && now.Sub(c.since) >= decorationSettle
		return false
	}
	if !whole {
		return false
	}
	// What the window painted, but where the area draws: GTK draws the
	// area apart, and what it painted under it never shows.
	cairoSave(cr)
	cairoSetOperator(cr, cairoOperatorSource)
	for i, b := range key.bands() {
		if c.bands[i] == 0 {
			continue
		}
		cairoSetSourceSurface(cr, c.bands[i], float64(b.X), float64(b.Y))
		cairoRectangle(cr, float64(b.X), float64(b.Y), float64(b.Width), float64(b.Height))
		cairoFill(cr)
	}
	cairoRestore(cr)
	gtkContainerPropagateDraw(win, s.w.box, cr)
	return true
}

// keepDecorations keeps what the window painted, after GTK painted it,
// when drawDecorations asked for it.
func (s *surface) keepDecorations(cr ptr) {
	c := &s.deco
	if !c.capture {
		return
	}
	c.capture = false
	target := cairoGetTarget(cr)
	for i, b := range c.key.bands() {
		if b.Width <= 0 || b.Height <= 0 {
			continue
		}
		scale := c.key.scale
		band := cairoImageSurfaceCreate(cairoFormatARGB32, b.Width*scale, b.Height*scale)
		if band == 0 {
			c.drop()
			return
		}
		cairoSurfaceSetDeviceScale(band, float64(scale), float64(scale))
		bc := cairoCreate(band)
		cairoSetSourceSurface(bc, target, -float64(b.X), -float64(b.Y))
		cairoSetOperator(bc, cairoOperatorSource)
		cairoPaint(bc)
		cairoDestroy(bc)
		c.bands[i] = band
	}
	c.kept = true
}

// drop lets go of the cache.
func (c *decorations) drop() {
	for i, b := range c.bands {
		if b != 0 {
			cairoSurfaceDestroy(b)
			c.bands[i] = 0
		}
	}
	c.capture, c.kept = false, false
}
