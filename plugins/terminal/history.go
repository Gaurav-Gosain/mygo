package terminal

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// ErrFullScreen is PrependHistory's error while a full-screen program's
// screen (the alternate screen) shows: the scrollback it would add to is
// the main screen's, which then does not show.
var ErrFullScreen = errors.New("terminal: a full-screen program's screen shows")

// PrependHistory adds rows above the oldest row of the scrollback, as a
// client of a server that keeps a session's terminal does when the view
// nears the top of the history it was sent and it fetches what came
// before. rows are VT sequences, row after row, each ended by "\r\n", or
// soft wrapped by running on to the last column, as History returns them.
// It returns how many rows the scrollback grew by, which is fewer than
// rows holds where the scrollback's limit lets go of the oldest.
//
// What the view shows stays where it is: scrolled into the history, it
// shows the same rows after, which are now further from the top; at the
// bottom, it stays there. The screen, the cursor and the modes programs
// set stay as they are, and output that comes later lands as it would
// have. The terminal is rebuilt to do this: it starts over with rows,
// then what it showed (Snapshot), so it costs as much as writing its
// whole scrollback again, and the images programs placed, which a
// snapshot does not carry, go. A search and a selection end.
func (t *Terminal) PrependHistory(rows []byte) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	t.mu.Lock()
	term := t.term
	if term == nil || t.closed {
		t.mu.Unlock()
		return 0, nil
	}
	if term.AltScreen() {
		t.mu.Unlock()
		return 0, ErrFullScreen
	}
	before := term.Scrollbar()
	bottom := term.AtBottom()
	snap := term.VT()
	title := term.Title()
	term.Reset()
	term.Write(rows)
	if !bytes.HasSuffix(rows, []byte("\n")) {
		// The last row ran on to its last column: what was the oldest
		// row starts below it.
		term.Write([]byte("\r\n"))
	}
	// The snapshot draws from the top of the screen: the rows still on
	// it, above the cursor, scroll into the scrollback first.
	if _, y := term.Cursor(); y > 0 {
		_, h := term.Size()
		term.Write(fmt.Appendf(nil, "\x1b[%dH%s", h, strings.Repeat("\n", y)))
	}
	term.Write(snap)
	if title != "" {
		// The snapshot leaves the title out, which Reset cleared.
		term.Write([]byte("\x1b]2;" + title + "\x1b\\"))
	}
	added := int(term.Scrollbar().Total) - int(before.Total)
	if bottom {
		term.ScrollToBottom()
	} else {
		term.ScrollToRow(int(before.Offset) + added)
	}
	t.events = events{}
	for _, m := range t.mirrors {
		m.started = false
	}
	t.mu.Unlock()
	t.redraw()
	return max(added, 0), nil
}

// nearTop reports whether OnNearTop is due: the view shows the scrollback
// within a screen (rows) of its top, at a row it was not told of yet. t.mu
// is held.
func (v *view) nearTop(t *Terminal, rows int) bool {
	sb := v.scrollbar
	if t.opts.OnNearTop == nil || sb.Offset >= uint64(rows) || t.term.AtBottom() || t.term.AltScreen() {
		v.toldTop = false
		return false
	}
	if v.toldTop && v.toldAt == sb.Offset {
		return false
	}
	v.toldTop, v.toldAt = true, sb.Offset
	return true
}
