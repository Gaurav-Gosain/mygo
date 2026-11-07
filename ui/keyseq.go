package ui

// nextKeySeq numbers a new key press, or text no press typed
// (InputEvent.Seq).
func (rt *engine) nextKeySeq() uint64 {
	rt.keySeq++
	return rt.keySeq
}

// textSeq is the number of text that comes: the last key press's, for the
// first text after it, as a key types its text after it goes down, else a
// number of its own, as for an input method's text.
func (rt *engine) textSeq() uint64 {
	if rt.pressSeq != 0 && !rt.pressTexted {
		rt.pressTexted = true
		return rt.pressSeq
	}
	return rt.nextKeySeq()
}

// pressOf notes that key, at physical, went down as press seq.
func (rt *engine) pressOf(key, physical Key, seq uint64) {
	if rt.pressSeqs == nil {
		rt.pressSeqs = map[Key]uint64{}
	}
	rt.pressSeqs[pressKey(key, physical)] = seq
}

// releaseOf returns the number of the press whose key, at physical, goes up,
// and 0 for one that went down elsewhere.
func (rt *engine) releaseOf(key, physical Key) uint64 {
	k := pressKey(key, physical)
	seq := rt.pressSeqs[k]
	delete(rt.pressSeqs, k)
	return seq
}

// pressKey names a key held down by its position when the backend gives
// one, as a release may come with other modifiers or another layout.
func pressKey(key, physical Key) Key {
	if physical != KeyUnknown {
		return physical
	}
	return key
}
