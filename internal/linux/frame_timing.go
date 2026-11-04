//go:build linux && (amd64 || arm64)

package linux

import (
	"math"
	"sync"
	"unsafe"
)

// GDK 3 paces a window's frame clock by the compositor only when it commits
// the window's buffer itself. A GtkGLArea's frames reach the compositor
// through eglSwapBuffers instead, with a swap interval of 0, so on Wayland
// GDK neither asks for a frame callback (gdk_window_impl_wayland_end_paint
// drops the commit it would ask one for) nor learns the refresh interval of
// the monitor. Its frame clock then runs on its own timer, at the interval
// of the last frame's timings, and those have none: GDK takes 1/60 s
// (FRAME_INTERVAL in gdk/gdkframeclockidle.c). A window that draws every
// frame draws 60 a second on a monitor that shows 120 or 144.
//
// paceFrameClock gives a frame whose timings have no refresh interval the
// interval of the monitor showing the area, which GDK's next cycle then
// waits for. GDK has no setter for it: the field is written where
// GdkFrameTimings keeps it in GTK 3.24 (gdk/gdkframeclockprivate.h), once
// the fields before it read as GDK's getters return them, and only when the
// write reads back through gdk_frame_timings_get_refresh_interval.
// Otherwise the frame clock keeps GDK's interval.
func (s *surface) paceFrameClock() {
	if !frameTimingOK() {
		return
	}
	clock := gtkWidgetGetFrameClock(s.area)
	if clock == 0 {
		return
	}
	t := gdkFrameClockGetCurrentTimings(clock)
	if t == 0 || gdkFrameTimingsGetRefreshInterval(t) != 0 {
		return // GDK knows it, from the compositor
	}
	hz := s.RefreshRate()
	if hz <= 0 {
		return
	}
	interval := int64(math.Round(1e6 / hz)) // µs
	if !timingsLayoutOK(t) {
		frameTiming.broken = true
		return
	}
	field := fieldAt[int64](t, timingsRefreshInterval)
	*field = interval
	if gdkFrameTimingsGetRefreshInterval(t) != interval {
		*field = 0
		frameTiming.broken = true
	}
}

// The offsets in GdkFrameTimings (GTK 3.24, 64-bit) of frame_counter,
// frame_time and refresh_interval, after a guint ref_count.
const (
	timingsFrameCounter    = 8
	timingsFrameTime       = 24
	timingsRefreshInterval = 56
)

var (
	gtkWidgetGetFrameClock            func(w ptr) ptr
	gdkFrameClockGetCurrentTimings    func(c ptr) ptr
	gdkFrameTimingsGetFrameCounter    func(t ptr) int64
	gdkFrameTimingsGetFrameTime       func(t ptr) int64
	gdkFrameTimingsGetRefreshInterval func(t ptr) int64

	frameTiming struct {
		once          sync.Once
		bound, broken bool
	}
)

// frameTimingOK reports whether paceFrameClock can give frames their
// refresh interval: GDK has the getters it checks the layout with, on a
// 64-bit system, and no frame has failed the check.
func frameTimingOK() bool {
	frameTiming.once.Do(func() {
		t, d := libGTK, libGDK
		frameTiming.bound = unsafe.Sizeof(uintptr(0)) == 8 &&
			bind(t, &gtkWidgetGetFrameClock, "gtk_widget_get_frame_clock") &&
			bind(d, &gdkFrameClockGetCurrentTimings, "gdk_frame_clock_get_current_timings") &&
			bind(d, &gdkFrameTimingsGetFrameCounter, "gdk_frame_timings_get_frame_counter") &&
			bind(d, &gdkFrameTimingsGetFrameTime, "gdk_frame_timings_get_frame_time") &&
			bind(d, &gdkFrameTimingsGetRefreshInterval, "gdk_frame_timings_get_refresh_interval")
	})
	return frameTiming.bound && !frameTiming.broken
}

// timingsLayoutOK reports whether the timings t hold their fields where
// paceFrameClock writes: the frame's counter and time read there as GDK's
// getters return them, and the refresh interval reads 0, as the getter
// said.
func timingsLayoutOK(t ptr) bool {
	ft := gdkFrameTimingsGetFrameTime(t)
	return ft != 0 && field[int64](t, timingsFrameTime) == ft &&
		field[int64](t, timingsFrameCounter) == gdkFrameTimingsGetFrameCounter(t) &&
		field[int64](t, timingsRefreshInterval) == 0
}

// fieldAt returns a pointer to the T at offset in the C struct at p.
func fieldAt[T any](p ptr, offset uintptr) *T {
	return (*T)(unsafe.Add(*(*unsafe.Pointer)(unsafe.Pointer(&p)), offset))
}
