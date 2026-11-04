//go:build windows && (amd64 || arm64)

package windows

import (
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// Pens and fingers come as WM_POINTER messages, which the surface leaves
// to DefWindowProc to make mouse messages of, as it does for windows that
// handle none: the surface reads their pressure from them, and the mouse
// messages that follow tell what device they come of.
var (
	procGetMessageExtraInfo = user32.NewProc("GetMessageExtraInfo")
	procGetPointerType      = user32.NewProc("GetPointerType")
	procGetPointerPenInfo   = user32.NewProc("GetPointerPenInfo")
	procGetPointerTouchInfo = user32.NewProc("GetPointerTouchInfo")
)

const (
	wmPointerUpdate = 0x0245
	wmPointerDown   = 0x0246
	wmPointerUp     = 0x0247

	ptTouch = 2
	ptPen   = 3

	penMaskPressure   = 0x1
	touchMaskPressure = 0x4

	// miWpSignature marks the extra information of mouse messages that
	// Windows makes of a pen's or a finger's (MI_WP_SIGNATURE), with
	// 0x80 set for a finger's.
	miWpSignature = 0xFF515700
	signatureMask = 0xFFFFFF00
)

// pointerInfo is POINTER_INFO.
type pointerInfo struct {
	Type, ID, FrameID, Flags uint32
	SourceDevice, Target     uintptr
	Pixel, Himetric          point
	PixelRaw, HimetricRaw    point
	Time, HistoryCount       uint32
	InputData                int32
	KeyStates                uint32
	PerformanceCount         uint64
	ButtonChangeType         uint32
}

// pointerPenInfo is POINTER_PEN_INFO.
type pointerPenInfo struct {
	Info                  pointerInfo
	Flags, Mask, Pressure uint32
	Rotation              uint32
	TiltX, TiltY          int32
}

// pointerTouchInfo is POINTER_TOUCH_INFO.
type pointerTouchInfo struct {
	Info                  pointerInfo
	Flags, Mask           uint32
	Contact, ContactRaw   rect
	Orientation, Pressure uint32
}

// The pointer messages' information, read on the main thread into these
// rather than into values of each message, which would go to the heap.
var (
	pointerType uint32
	penInfo     pointerPenInfo
	touchInfo   pointerTouchInfo
)

// pointerPressure returns how hard the pen or the finger of WM_POINTER's
// pointer id presses, from 0 to 1, or 0 where it does not tell, as a
// mouse's.
func pointerPressure(id uint32) float64 {
	if !has(procGetPointerType) {
		return 0
	}
	if r, _, _ := procGetPointerType.Call(uintptr(id), uintptr(unsafe.Pointer(&pointerType))); r == 0 {
		return 0
	}
	switch pointerType {
	case ptPen:
		if r, _, _ := procGetPointerPenInfo.Call(uintptr(id), uintptr(unsafe.Pointer(&penInfo))); r != 0 && penInfo.Mask&penMaskPressure != 0 {
			return float64(penInfo.Pressure) / 1024
		}
	case ptTouch:
		if r, _, _ := procGetPointerTouchInfo.Call(uintptr(id), uintptr(unsafe.Pointer(&touchInfo))); r != 0 && touchInfo.Mask&touchMaskPressure != 0 {
			return float64(touchInfo.Pressure) / 1024
		}
	}
	return 0
}

// mouseDevice returns the device of the mouse message being handled, as
// Windows marks those it makes of a pen's or a finger's, with the pressure
// that the pointer message before it told.
func (s *surface) mouseDevice() (platform.PointerKind, float64) {
	extra, _, _ := procGetMessageExtraInfo.Call()
	switch {
	case uint32(extra)&signatureMask != miWpSignature:
		return platform.PointerMouse, 0
	case extra&0x80 != 0:
		return platform.PointerTouch, s.pressure
	}
	return platform.PointerPen, s.pressure
}
