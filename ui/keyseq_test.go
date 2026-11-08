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

// TestTextTypedBehindASubmitKeepsItsPlace: Enter in a single-line field
// submits it, which the next frame reads, as a palette that closes and
// gives the focus to a pane. Text typed after the Enter, before that frame,
// goes to the pane, never into the field.
func TestTextTypedBehindASubmitKeepsItsPlace(t *testing.T) {
	field := ""
	var pane []string
	toPane := false
	tt := coreNewTester(func(c *context) {
		p := coreBox(c).Key("pane").Size(100, 50).Focusable().HandleInput(func(ev InputEvent) bool {
			switch ev.Kind {
			case InputText:
				pane = append(pane, ev.Text)
				return true
			case InputKeyDown:
				return true
			}
			return false
		})
		if toPane {
			p.Focus()
			return
		}
		in := textInputBase(c, &field, false).Size(200, 40)
		in.Focus()
		if in.Submitted() {
			toPane = true
			field = ""
		}
	}, 300, 100)
	tt.Frame()
	tt.Frame()
	tt.Type("tests")
	tt.Frame()
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyEnter)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeyEnter)})
	for _, r := range []Key{KeyT, KeyT, KeyY} {
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(r)})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.TextInput, Text: string(rune('a' + r - KeyA))})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(r)})
	}
	for range 4 {
		tt.Frame()
	}
	if !toPane {
		t.Fatalf("the Enter never submitted the field")
	}
	if got := strings.Join(pane, ""); got != "tty" {
		t.Fatalf("the pane has %q, want tty", got)
	}
}

// TestAModifierAloneHoldsNothing: a key down with no key, as Shift alone
// comes from some senders, matches no shortcut. Text typed after it goes
// to the field at once, with no frame between: else every Shift typed
// held the text behind it until a frame came.
func TestAModifierAloneHoldsNothing(t *testing.T) {
	var pane []string
	tt := coreNewTester(func(c *context) {
		// A terminal: it takes the keys it has a name for, and text.
		coreBox(c).Key("pane").Size(100, 50).Focusable().Focus().HandleInput(func(ev InputEvent) bool {
			switch ev.Kind {
			case InputText:
				pane = append(pane, ev.Text)
				return true
			case InputKeyDown:
				return ev.Key != KeyUnknown
			}
			return false
		})
	}, 300, 100)
	tt.Frame()
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Mods: platform.Modifiers(Shift)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.TextInput, Text: ">"})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Mods: platform.Modifiers(Shift)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeySlash)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.TextInput, Text: "/"})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeySlash)})
	if n := len(tt.rt.behind); n != 0 || strings.Join(pane, "") != ">/" {
		t.Fatalf("%d events wait behind a Shift alone and the pane has %q, want none and >/", n, strings.Join(pane, ""))
	}
}

// TestTextTypedBehindADialogSubmitGoesBackToTheOpener: a dialog with a
// field, as a command palette, opened from a pane. Enter submits the field
// and the app closes the dialog in the frame that reads it. The focus goes
// back to the pane only in the frame after, when the dialog is gone. Text
// typed after the Enter must reach the pane: replayed in the frame that
// read the Enter, it went into the closing field and was lost with it (the
// first letter of a line typed after a palette jump, b10).
func TestTextTypedBehindADialogSubmitGoesBackToTheOpener(t *testing.T) {
	field := ""
	var pane []string
	open, first, back := false, true, false
	tt := coreNewTester(func(c *context) {
		p := coreBox(c).Key("pane").Size(100, 50).Focusable().HandleInput(func(ev InputEvent) bool {
			switch ev.Kind {
			case InputText:
				pane = append(pane, ev.Text)
				return true
			case InputKeyDown:
				return true
			}
			return false
		})
		if first || back {
			// The app gives the pane the keys back once the dialog is
			// gone, as guios's key owner does.
			p.Focus()
			first, back = false, false
		}
		coreDialogBase(c, &open, func(_, panel *node) {
			in := textInputBase(c, &field, false).Size(200, 40).AutoFocus()
			if in.Submitted() {
				open, back = false, true
			}
		})
	}, 300, 100)
	tt.Frame()
	open = true
	tt.Frame()
	tt.Frame()
	tt.Type("tests")
	tt.Frame()
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyEnter)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeyEnter)})
	for _, r := range []Key{KeyT, KeyT, KeyY} {
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(r)})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.TextInput, Text: string(rune('a' + r - KeyA))})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(r)})
	}
	for range 4 {
		tt.Frame()
	}
	if open {
		t.Fatalf("the Enter never closed the dialog")
	}
	if got := strings.Join(pane, ""); got != "tty" {
		t.Fatalf("the pane has %q, want tty (field %q)", got, field)
	}
}

// TestTextAnEditorTakesBehindAWaitingKeyKeepsItsPlace: a letter typed in a
// list waits for the list (routed where it was typed), and the letters
// typed after it, once a field has the focus, come by the field's text
// client at once, as AppKit's insertText gives them. The list writes its
// letter into the field's string when it reads it, in the frame the field
// took the others. Both stay, the list's letter first; the app's string
// won, and the field lost the letters typed after it (guios b10).
func TestTextAnEditorTakesBehindAWaitingKeyKeepsItsPlace(t *testing.T) {
	s := ""
	inField, toEnd := false, false
	tt := coreNewTester(func(c *context) {
		// The field first, as guios builds the reply box before the list.
		in := textInput(c, &s, false).Size(200, 30)
		if toEnd {
			// The caret after the list's text, as guios's reply box puts it.
			toEnd = false
			n := len([]rune(s))
			in.SetTextSelection(n, n)
		}
		l := coreBox(c).Key("list").Size(100, 50).Focusable()
		if !inField {
			l.Focus()
		}
		if l.Shortcut(0, KeySpace) {
			inField = true
			in.Focus()
		}
		if l.Shortcut(0, KeyM) {
			s += "m"
			toEnd = true
		}
	}, 300, 120)
	defer tt.rt.close()
	tt.Frame()
	press := func(k Key) {
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(k)})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(k)})
	}
	// Space and m reach the list before a frame: Space gives the field the
	// focus, the m waits behind it.
	press(KeySpace)
	press(KeyM)
	tt.rt.runFrame()
	// The field has the focus: m and o come by its client, while the
	// list's m still waits.
	cl := tt.h.ime.Client
	for _, k := range []struct {
		key  Key
		text string
	}{{KeyM, "m"}, {KeyO, "o"}} {
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(k.key)})
		cl.ReplaceText(nil, k.text)
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(k.key)})
	}
	for range 4 {
		tt.Frame()
	}
	if s != "mmo" {
		t.Fatalf("the field has %q, want mmo", s)
	}
}

// TestTextAnEditorTakesBehindTwoWaitingKeysKeepsItsPlace: two letters typed
// in a list wait for it, one a frame, and a letter typed after them, once a
// field has the focus, comes by the field's text client. It must wait
// behind the second, as a key down does: the client took it at once, and
// the field showed "mom" for "mmo" (guios b10).
func TestTextAnEditorTakesBehindTwoWaitingKeysKeepsItsPlace(t *testing.T) {
	s := ""
	inField, toEnd := false, false
	tt := coreNewTester(func(c *context) {
		in := textInput(c, &s, false).Size(200, 30)
		if toEnd {
			// The caret after the list's text, as guios's reply box puts it.
			toEnd = false
			n := len([]rune(s))
			in.SetTextSelection(n, n)
		}
		l := coreBox(c).Key("list").Size(100, 50).Focusable()
		if !inField {
			l.Focus()
		}
		if l.Shortcut(0, KeySpace) {
			inField = true
			in.Focus()
		}
		if l.Shortcut(0, KeyM) {
			s += "m"
			toEnd = true
		}
	}, 300, 120)
	defer tt.rt.close()
	tt.Frame()
	press := func(k Key) {
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(k)})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(k)})
	}
	press(KeySpace)
	press(KeyM)
	press(KeyM)
	tt.rt.runFrame()
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyO)})
	tt.h.ime.Client.ReplaceText(nil, "o")
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeyO)})
	for range 4 {
		tt.Frame()
	}
	if s != "mmo" {
		t.Fatalf("the field has %q, want mmo", s)
	}
}
