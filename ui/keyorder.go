package ui

import "github.com/Gaurav-Gosain/mygo/internal/platform"

// heldEvent is input that came while a key waited for a frame (keysWait),
// with the element that had the focus as it came.
type heldEvent struct {
	ev    platform.SurfaceEvent
	focus uint64
}

// keysWait reports whether a key pressed waits for the next frame: a key a
// shortcut reads then, a second key for an element that took one, or a key
// that acts (actedOn), whose outcome the view shows then.
func (rt *engine) keysWait() bool {
	return len(rt.keys) > 0 || len(rt.retried) > 0 || rt.keyActs
}

// actedOn notes that key k went down: a key that types no text acts (Enter
// submits a field, Escape closes a panel, a chord switches the view), and
// the view moves the focus as it builds the next frame. What is typed after
// it waits for that frame, then goes where the frame put the focus. Keys
// replayed set nothing: the replay delivers them all after one frame.
func (rt *engine) actedOn(k keyEvent) {
	if rt.replaying || typesText(k) || k.key == KeyUnknown {
		// A modifier alone, as some senders give it, acts on nothing.
		return
	}
	rt.keyActs = true
	rt.requestFrame()
}

// holdBehind keeps input that comes while a key waits behind that key, and
// reports whether it did. Text and keys typed after a key that waits would
// otherwise reach the focus at once, before the key moves it: an Escape
// that closes a field, and the text typed after it into the field. The
// frame that routes the key replays what is held (replayBehind). A key's
// own text and release, which come while nothing is held, go at once, as
// before.
func (rt *engine) holdBehind(ev platform.SurfaceEvent) bool {
	if rt.replaying {
		return false
	}
	switch ev.Kind {
	case platform.KeyPressed, platform.SurfaceCommand:
		if len(rt.behind) == 0 && !rt.keysWait() {
			return false
		}
	case platform.KeyReleased, platform.ModifiersChanged, platform.TextInput, platform.PointerDown, platform.PointerUp:
		if len(rt.behind) == 0 {
			return false
		}
	case platform.SurfaceBlur:
		// Typed while the window had the keyboard: it goes now, while
		// the focus takes text still.
		rt.replayBehind(true)
		return false
	default:
		return false
	}
	rt.behind = append(rt.behind, heldEvent{ev, rt.focused})
	rt.requestFrame()
	return true
}

// holdTyped keeps text an editor's client was given (AppKit's insertText
// with an editor focused) behind the keys held, as text, and reports
// whether it did. The host gives it at once, past rt.event, after the key
// down holdBehind kept: else the editor took it before the keys typed
// first (guios b10). Replayed, it goes in by the focus then.
func (rt *engine) holdTyped(text string) bool {
	if rt.replaying || len(rt.behind) == 0 || text == "" {
		return false
	}
	rt.behind = append(rt.behind, heldEvent{platform.SurfaceEvent{Kind: platform.TextInput, Text: text}, rt.focused})
	rt.requestFrame()
	return true
}

// replayBehind delivers the input held behind keys that waited, in the
// order it came, to the focus the frames since gave. It stops before a key
// down, a command or a press while a key waits again, for the next frame;
// all replays everything.
func (rt *engine) replayBehind(all bool) {
	if len(rt.behind) == 0 || rt.replaying {
		return
	}
	rt.replaying = true
	defer func() { rt.replaying, rt.replayFocus, rt.replayUntexted = false, 0, false }()
	for len(rt.behind) > 0 {
		h := rt.behind[0]
		switch h.ev.Kind {
		case platform.KeyPressed, platform.SurfaceCommand, platform.PointerDown:
			if !all && rt.keysWait() {
				return
			}
		}
		rt.behind = rt.behind[1:]
		rt.replayFocus = h.focus
		rt.replayUntexted = h.ev.Kind == platform.KeyPressed && !textFollows(rt.behind)
		rt.event(h.ev)
	}
	rt.behind = nil
}

// textFollows reports whether the held input begins with text before the
// next key down: the text of the key down replaying.
func textFollows(behind []heldEvent) bool {
	for _, h := range behind {
		switch h.ev.Kind {
		case platform.TextInput:
			return true
		case platform.KeyPressed, platform.SurfaceCommand:
			return false
		}
	}
	return false
}

// keptWhereTyped puts a replayed key that typed no text, as a letter typed
// into a list, on the queue for the element it was typed in, when the
// focus has moved since to one that would take it as text, which never
// comes: the list's shortcut for the letter reads it. It reports whether
// it did.
func (rt *engine) keptWhereTyped(k keyEvent) bool {
	if !rt.replaying || !rt.replayUntexted || !typesText(k) || rt.typedIn() == rt.focused {
		return false
	}
	rt.keys = append(rt.keys, pressedKey{k, rt.pressSeq, rt.typedIn()})
	rt.requestFrame()
	return true
}

// typedIn returns the element a key put on the queue was typed in: the
// focus, or for a key replayed from behind another, the focus as it came,
// while that element is still there. routeKeys routes it by that element.
func (rt *engine) typedIn() uint64 {
	if id := rt.replayFocus; rt.replaying && id != 0 && rt.states[id] != nil {
		return id
	}
	return rt.focused
}
