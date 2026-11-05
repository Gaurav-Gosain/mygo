package terminal

import (
	"bytes"
	"testing"
	"time"

	"github.com/Gaurav-Gosain/mygo/ui"
)

// TestBlinkStops checks that a blinking cursor stops after the desktop's
// caret blink timeout, as GTK's does, so that a terminal left alone draws
// no frame and sets no timer; and that a key, output that moves the cursor
// and the focus coming back each make it blink again.
//
// Ways it can go wrong, each checked below:
//   - the cursor never blinks, or blinks on after the timeout;
//   - it stops in its off phase and stays hidden;
//   - a frame or timer keeps coming after the timeout;
//   - a key, output that moves the cursor or the focus does not restart it;
//   - a negative timeout (GTK's G_MAXINT) stops it all the same.
func TestBlinkStops(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	frames := 0
	tt := ui.NewTester(func(c *ui.Context) {
		frames++
		View(c, term).Fill().AutoFocus()
	}, 400, 200)
	tt.SetPreferences(ui.Preferences{TextScale: 1, CaretBlinkTimeout: 3300 * time.Millisecond})
	term.Feed([]byte("$ "))
	tt.Frame()
	// The blink restarts on the frame that saw the cursor move.
	tt.Advance(10 * time.Millisecond)
	shown := tt.Image()
	tt.Advance(blinkPeriod)
	if bytes.Equal(shown.Pix, tt.Image().Pix) {
		t.Fatal("the cursor did not blink")
	}
	tt.Advance(blinkPeriod)
	if !bytes.Equal(shown.Pix, tt.Image().Pix) {
		t.Fatal("the cursor did not come back on")
	}

	settled := func(what string) {
		t.Helper()
		// 3.3 s ends in the off phase that starts at 3 s: without the
		// toggle that shows it the cursor would stay hidden.
		tt.Advance(4 * time.Second)
		if !bytes.Equal(shown.Pix, tt.Image().Pix) {
			t.Errorf("%s: the cursor is not on after the timeout", what)
		}
		n := frames
		tt.Advance(time.Hour)
		if frames != n || tt.FramePending() {
			t.Errorf("%s: %d frames in an hour after the timeout", what, frames-n)
		}
	}
	settled("left alone")

	restarted := func(what string) {
		t.Helper()
		tt.Advance(10 * time.Millisecond)
		shown = tt.Image()
		tt.Advance(blinkPeriod)
		if bytes.Equal(shown.Pix, tt.Image().Pix) {
			t.Errorf("%s did not make it blink again", what)
		}
	}
	tt.Type("a")
	restarted("a key")
	settled("after a key")

	term.Feed([]byte("\r\n$ "))
	tt.Frame()
	restarted("output that moves the cursor")
	settled("after output")

	tt.SetFocused(false)
	tt.SetFocused(true)
	restarted("the focus")
	settled("after the focus")

	tt.SetPreferences(ui.Preferences{TextScale: 1, CaretBlinkTimeout: -1})
	tt.Type("b")
	tt.Advance(time.Minute)
	n := frames
	tt.Advance(2 * blinkPeriod)
	if frames == n {
		t.Error("a negative timeout stopped the blinking")
	}
}
