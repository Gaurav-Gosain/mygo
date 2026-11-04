package terminal

import (
	"strings"
	"testing"

	"github.com/Gaurav-Gosain/mygo/plugins/terminal/internal/vt"
	"github.com/Gaurav-Gosain/mygo/ui"
)

// viewportText is the text of the rows the viewport shows, and where it is.
func viewportText(term *Terminal) (string, vt.Scrollbar, bool) {
	term.mu.Lock()
	defer term.mu.Unlock()
	_, rows := term.term.Size()
	var b strings.Builder
	for y := range rows {
		text, _ := term.term.RowText(y)
		b.WriteString(strings.TrimRight(string(text), " \x00"))
		b.WriteByte('\n')
	}
	return b.String(), term.term.Scrollbar(), term.term.AtBottom()
}

func TestPrependHistory(t *testing.T) {
	loadLib(t)
	newTerm := func() *Terminal {
		term, err := New(Options{Conn: newPipe()})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { term.Close() })
		term.Resize(20, 5)
		return term
	}
	// whole is the session as a server keeps it: older rows, one of them
	// wider than the screen, then what the client was sent.
	var older, recent strings.Builder
	for i := range 30 {
		older.WriteString("old " + itoa(i) + " \x1b[31mred\x1b[0m\r\n")
		if i == 20 {
			older.WriteString(strings.Repeat("w", 30) + "\r\n") // soft wrapped at 20
		}
	}
	for i := range 12 {
		recent.WriteString("new " + itoa(i) + "\r\n")
	}
	recent.WriteString("\x1b]2;the title\x07prompt % \x1b[?2004h")
	whole := newTerm()
	whole.Feed([]byte(older.String() + recent.String()))
	part := newTerm()
	part.Feed([]byte(recent.String()))

	for _, scrolled := range []bool{false, true} {
		term := newTerm()
		term.Feed(part.Snapshot())
		term.Feed([]byte("\x1b]2;the title\x07"))
		if scrolled {
			term.mu.Lock()
			term.term.ScrollBy(-3)
			term.mu.Unlock()
		}
		shown, before, _ := viewportText(term)
		n, err := term.PrependHistory(whole.historyAbove(t, part))
		if err != nil {
			t.Fatal(err)
		}
		if n != 32 { // 30 rows and the wrapped one's two
			t.Errorf("scrolled %v: added %d rows, want 32", scrolled, n)
		}
		got, after, bottom := viewportText(term)
		if got != shown {
			t.Errorf("scrolled %v: the viewport shows\n%s\nwant\n%s", scrolled, got, shown)
		}
		if bottom == scrolled {
			t.Errorf("scrolled %v: at the bottom %v", scrolled, bottom)
		}
		if after.Total != before.Total+32 {
			t.Errorf("scrolled %v: scrollbar %+v after %+v", scrolled, after, before)
		}
		// The text, the cursor and the modes are the whole session's:
		// what comes next lands alike.
		term.Feed([]byte("ls"))
		w := newTerm()
		w.Feed([]byte(older.String() + recent.String() + "ls"))
		if term.Text() != w.Text() {
			t.Errorf("scrolled %v: text\n%q, want\n%q", scrolled, term.Text(), w.Text())
		}
		term.mu.Lock()
		paste, title := term.term.Mode(vt.ModeBracketedPaste), term.term.Title()
		term.mu.Unlock()
		if !paste || title != "the title" {
			t.Errorf("scrolled %v: bracketed paste %v, title %q", scrolled, paste, title)
		}
	}

	full := newTerm()
	full.Feed([]byte("\x1b[?1049h"))
	if _, err := full.PrependHistory([]byte("x\r\n")); err != ErrFullScreen {
		t.Errorf("full screen: err %v", err)
	}
}

// historyAbove returns the rows of t's history above those part holds, as
// a server sends them: History's rows, less part's.
func (t *Terminal) historyAbove(tb *testing.T, part *Terminal) []byte {
	tb.Helper()
	all, mine := t.History(), part.History()
	if !strings.HasSuffix(string(all), string(mine)) {
		tb.Fatalf("history %q does not end with %q", all, mine)
	}
	return all[:len(all)-len(mine)]
}

// TestOnNearTop: scrolling up to within a screen of the top asks for older
// rows, once a row, and the rows PrependHistory adds keep the view where it
// is, out of reach of the top.
func TestOnNearTop(t *testing.T) {
	loadLib(t)
	var asked int
	term, err := New(Options{Conn: newPipe(), OnNearTop: func() { asked++ }})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 40 {
		b.WriteString("line " + itoa(i) + "\r\n")
	}
	term.Feed([]byte(b.String()))
	tt.Frame()
	if asked != 0 {
		t.Fatalf("asked %d times at the bottom", asked)
	}
	_, rows := term.Size()
	term.mu.Lock()
	term.term.ScrollToRow(rows) // a screen from the top: not yet
	term.mu.Unlock()
	term.redraw()
	tt.Frame()
	if asked != 0 {
		t.Fatalf("asked %d times a screen from the top", asked)
	}
	tt.Key(ui.Shift, ui.KeyPageUp)
	tt.Frame()
	if asked != 1 {
		t.Fatalf("asked %d times near the top, want 1", asked)
	}
	tt.Frame()
	if asked != 1 {
		t.Fatalf("asked %d times after a frame that did not move, want 1", asked)
	}
	shown, _, _ := viewportText(term)
	var older strings.Builder
	for i := range 50 {
		older.WriteString("older " + itoa(i) + "\r\n")
	}
	if _, err := term.PrependHistory([]byte(older.String())); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got, _, _ := viewportText(term); got != shown {
		t.Errorf("after the rows came, the view shows\n%s\nwant\n%s", got, shown)
	}
	if asked != 1 {
		t.Errorf("asked %d times once the rows came, want 1", asked)
	}
}
