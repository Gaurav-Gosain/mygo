package ui

import (
	"strings"
	"testing"

	"github.com/Gaurav-Gosain/mygo/internal/platform"
)

// TestKeySeqKeepsADoubleLetter: each press of a key has a sequence number of
// its own, which its key down, its text and its release share, in a
// handler and in a shortcut. An app that drops a copy of a key by that
// number never drops the second of two equal letters typed.
func TestKeySeqKeepsADoubleLetter(t *testing.T) {
	toList := false
	type seen struct {
		kind InputKind
		seq  uint64
	}
	var term []seen
	var list []uint64
	tt := coreNewTester(func(c *context) {
		box := coreBox(c).Key("term").Size(100, 50).Focusable().HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputKeyDown || ev.Kind == InputText || ev.Kind == InputKeyUp {
				term = append(term, seen{ev.Kind, ev.Seq})
			}
			return true
		})
		if !toList {
			box.Focus()
		}
		l := coreBox(c).Key("list").Size(100, 50).Focusable()
		if toList {
			l.Focus()
		}
		if l.Shortcut(0, KeyV) {
			list = append(list, c.ShortcutSeq())
		}
	}, 200, 100)
	tt.Frame()

	tt.TypeKey(0, KeyV, "v")
	tt.TypeKey(0, KeyV, "v")
	if len(term) != 6 {
		t.Fatalf("the handler saw %d events, want 6: %v", len(term), term)
	}
	first, second := term[0].seq, term[3].seq
	if first == 0 || second == 0 || first == second {
		t.Fatalf("two presses of v have seq %d and %d, want two numbers", first, second)
	}
	for i, s := range term {
		if want := []uint64{first, second}[i/3]; s.seq != want {
			t.Fatalf("event %d (%v) has seq %d, want the press's %d", i, s.kind, s.seq, want)
		}
	}

	// Two presses as the focus moves to the list, before a frame shows it.
	toList = true
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyV)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeyV)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyV)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeyV)})
	tt.Frame()
	tt.Frame()
	tt.Key(0, KeyV)
	tt.Key(0, KeyV)
	seqs := map[uint64]bool{first: true, second: true}
	for _, s := range term[6:] {
		if s.kind == InputKeyDown {
			seqs[s.seq] = true
		}
	}
	for _, s := range list {
		if s == 0 || seqs[s] {
			t.Fatalf("the list got seq %d, which is 0 or a press the handler had: list %v, handler %v", s, list, term)
		}
		seqs[s] = true
	}
	if got := len(seqs); got != 6 {
		t.Fatalf("six presses have %d sequence numbers: list %v, handler %v", got, list, term)
	}
}

// TestKeyWaitingKeepsItsFocus types two keys before a frame into a list
// that reads them by shortcut, whose first key moves the focus to a text
// input. An element takes one key a frame, so the second waits; it still
// goes to the list it was typed in, not to the input that has the focus by
// the next frame, where nothing took it and it was lost.
func TestKeyWaitingKeepsItsFocus(t *testing.T) {
	text := ""
	var got []Key
	toField := false
	tt := coreNewTester(func(c *context) {
		list := coreBox(c).Key("list").Size(100, 50).Focusable()
		if !toField {
			list.Focus()
		}
		for _, k := range []Key{KeyM, KeyO} {
			if list.Shortcut(0, k) {
				got = append(got, k)
				toField = true
			}
		}
		in := coreTextAreaBase(c, &text).Size(200, 40)
		if toField {
			in.Focus()
		}
	}, 300, 100)
	tt.Frame()
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyM)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeyM)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyM)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeyM)})
	tt.Frame()
	tt.Frame()
	if len(got) != 2 {
		t.Fatalf("the list got %v, want both m", got)
	}
}

// TestTextTypedBehindAWaitingKeyKeepsItsPlace: Escape in a field is the
// window's shortcut, read in the next frame, and it moves the focus from
// the field to a pane that reads keys and text as they come. Text typed
// after the Escape, before that frame, goes where the Escape sent the focus,
// never into the field: keys and text leave the queue in the order they
// came.
func TestTextTypedBehindAWaitingKeyKeepsItsPlace(t *testing.T) {
	field := ""
	var pane []string
	toPane := false
	tt := coreNewTester(func(c *context) {
		if c.Shortcut(0, KeyEscape) {
			toPane = true
		}
		p := coreBox(c).Key("pane").Size(100, 50).Focusable().HandleInput(func(ev InputEvent) bool {
			switch ev.Kind {
			case InputText:
				pane = append(pane, ev.Text)
				return true
			case InputKeyDown:
				return ev.Key != KeyEscape
			}
			return false
		})
		in := textInputBase(c, &field, false).Size(200, 40)
		if toPane {
			p.Focus()
		} else {
			in.Focus()
		}
	}, 300, 100)
	tt.Frame()
	tt.Frame()
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyEscape)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeyEscape)})
	for _, r := range []Key{KeyT, KeyT, KeyY} {
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(r)})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.TextInput, Text: string(rune('a' + r - KeyA))})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(r)})
	}
	for range 4 {
		tt.Frame()
	}
	if !toPane {
		t.Fatalf("the Escape never reached the window's shortcut")
	}
	if field != "" || strings.Join(pane, "") != "tty" {
		t.Fatalf("the field has %q and the pane %q, want the pane to have all of tty", field, strings.Join(pane, ""))
	}
}
