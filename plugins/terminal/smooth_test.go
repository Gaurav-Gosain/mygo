package terminal

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Gaurav-Gosain/mygo/ui"
)

// scrolled returns a terminal with 100 lines of history in a Tester, at
// the bottom, and how to read where its viewport is: its top row, and how
// far the content is pulled down past it, in DIPs.
func scrolled(t *testing.T) (*Terminal, *ui.Tester, func() (uint64, float32), float32) {
	t.Helper()
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { term.Close() })
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 100 {
		b.WriteString("line " + itoa(i) + "\r\n")
	}
	term.Feed([]byte(b.String()))
	tt.Frame()
	at := func() (uint64, float32) {
		term.mu.Lock()
		defer term.mu.Unlock()
		return term.term.Scrollbar().Offset, term.v.pull
	}
	return term, tt, at, float32(term.v.cellH) / term.v.scale
}

// TestTouchpadScrollsByPixels: a touchpad's moves scroll the scrollback
// by pixels: half a row pulls the content down half a row, showing the
// row above, and the viewport moves by whole rows only.
func TestTouchpadScrollsByPixels(t *testing.T) {
	_, tt, at, ch := scrolled(t)
	bottom, _ := at()
	tt.Scroll(100, 100, 0, -ch/2)
	tt.Advance(100 * time.Millisecond)
	if off, pull := at(); off != bottom || pull != ch/2 {
		t.Errorf("half a row up: offset %d, pull %v; want %d, %v", off, pull, bottom, ch/2)
	}
	tt.Scroll(100, 100, 0, -ch)
	tt.Advance(100 * time.Millisecond)
	if off, pull := at(); off != bottom-1 || pull != ch/2 {
		t.Errorf("a row and a half up: offset %d, pull %v; want %d, %v", off, pull, bottom-1, ch/2)
	}
	tt.Scroll(100, 100, 0, 3*ch)
	tt.Advance(100 * time.Millisecond)
	if off, pull := at(); off != bottom || pull != 0 {
		t.Errorf("back down past the bottom: offset %d, pull %v; want %d, 0", off, pull, bottom)
	}
}

// TestTouchpadMovesEveryFrame: moves that come once a frame scroll every
// frame by about as much, though each lands just before or after a frame.
func TestTouchpadMovesEveryFrame(t *testing.T) {
	_, tt, at, ch := scrolled(t)
	pos := func() float32 {
		off, pull := at()
		return float32(off)*ch - pull
	}
	last := pos()
	var moves []float32
	for i := range 30 {
		// A move every 8 ms, 2 DIPs each; frames come at 60 Hz.
		tt.Scroll(100, 100, 0, -2)
		tt.Advance(8 * time.Millisecond)
		if i%2 == 1 {
			p := pos()
			moves = append(moves, last-p)
			last = p
		}
	}
	for i, m := range moves[2:] {
		if m < 2 || m > 6 {
			t.Errorf("frame %d moved %v DIPs, want about 4: %v", i+2, m, moves)
			break
		}
	}
}

// TestNotchGlides: a wheel's notch glides its three rows, rather than
// jumping them, and lands on a row.
func TestNotchGlides(t *testing.T) {
	_, tt, at, _ := scrolled(t)
	bottom, _ := at()
	tt.Wheel(100, 100, 0, -hostNotch())
	tt.Advance(notchGlide / 4)
	if off, pull := at(); off == bottom-3 && pull == 0 || off == bottom && pull == 0 {
		t.Errorf("a quarter of the way through the glide: offset %d, pull %v, not on the way", off, pull)
	}
	tt.Advance(notchGlide)
	if off, pull := at(); off != bottom-3 || pull != 0 {
		t.Errorf("after the glide: offset %d, pull %v; want %d, 0", off, pull, bottom-3)
	}
}

// TestFlickGoesOn: on Linux, a touchpad's flick goes on after the fingers
// lift, which GTK tells with a scroll without distance, slowing until it
// stops; elsewhere the system sends the momentum itself.
func TestFlickGoesOn(t *testing.T) {
	term, tt, at, _ := scrolled(t)
	bottom, _ := at()
	for range 10 {
		tt.Scroll(100, 100, 0, -16) // 2000 DIPs a second
		tt.Advance(8 * time.Millisecond)
	}
	lifted, _ := at()
	tt.Wheel(100, 100, 0, 0) // the fingers lift
	tt.Advance(2 * time.Second)
	off, _ := at()
	if runtime.GOOS != "linux" {
		if off != lifted && off+1 != lifted {
			t.Errorf("without momentum: offset %d after lifting at %d", off, lifted)
		}
		return
	}
	if off+5 > lifted {
		t.Errorf("the flick stopped as the fingers lifted: offset %d, %d at the lift, %d at the bottom", off, lifted, bottom)
	}
	if term.v.smooth.moving() {
		t.Error("the flick still moves after 2 s")
	}
}

// TestTypingStopsTheScroll: a key typed during a flick goes back to the
// bottom and stays there.
func TestTypingStopsTheScroll(t *testing.T) {
	_, tt, at, _ := scrolled(t)
	bottom, _ := at()
	for range 10 {
		tt.Scroll(100, 100, 0, -16)
		tt.Advance(8 * time.Millisecond)
	}
	tt.Wheel(100, 100, 0, 0)
	tt.Advance(50 * time.Millisecond)
	tt.Type("x")
	tt.Advance(time.Second)
	if off, pull := at(); off != bottom || pull != 0 {
		t.Errorf("after typing: offset %d, pull %v; want %d, 0", off, pull, bottom)
	}
}
