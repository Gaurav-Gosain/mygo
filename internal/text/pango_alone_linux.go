//go:build linux && (amd64 || arm64)

package text

// shapedSetKey names the glyphs of a font that the OpenType features
// shaping applies with a feature list may change or start at.
type shapedSetKey struct {
	font     uintptr // the PangoFont
	features string
}

// The OpenType tables HarfBuzz shapes with.
const (
	tagGSUB = 'G'<<24 | 'S'<<16 | 'U'<<8 | 'B'
	tagGPOS = 'G'<<24 | 'P'<<16 | 'O'<<8 | 'S'
)

// hbDefaultFeatures are the features HarfBuzz applies to horizontal text
// of the default shaper (hb-ot-shape.cc), but for mark and mkmk, which
// only attach marks. Pango adds none of its own to them.
var hbDefaultFeatures = [...]string{
	"abvm", "blwm", "ccmp", "locl", "rlig", "rvrn", "ltra", "ltrm",
	"calt", "clig", "curs", "dist", "kern", "liga", "rclt",
}

// hbGlyphClassMark is GDEF's class of mark glyphs.
const hbGlyphClassMark = 3

// alone returns the glyph of r in f, and whether shaping a text in f with
// features gives r that glyph and advance wherever it stands: the font has
// the glyph, it is no mark, and no lookup of a feature HarfBuzz applies
// can match at it. Lookups only match at a glyph in their input: text of
// such glyphs alone shapes as each of them does alone.
func (e *pangoEngine) alone(f *Font, features string, r rune) (uint32, bool) {
	l := &pangoLib
	if f == nil || l.fontGetHBFont == nil || l.hbNominalGlyph == nil || l.hbFontGetFace == nil ||
		l.hbSetCreate == nil || l.hbSetDestroy == nil || l.hbSetHas == nil || l.hbCollectLookups == nil ||
		l.hbLookupCollectGlyphs == nil || l.hbGlyphClass == nil || l.hbSetNext == nil {
		return 0, false
	}
	hb := l.fontGetHBFont(f.native)
	if hb == 0 {
		return 0, false
	}
	var id uint32
	if l.hbNominalGlyph(hb, r, &id) == 0 || id == 0 {
		return 0, false
	}
	face := l.hbFontGetFace(hb)
	if l.hbGlyphClass(face, id) == hbGlyphClassMark {
		return 0, false
	}
	key := shapedSetKey{f.native, features}
	set, ok := e.shapedSets[key]
	if !ok {
		set = e.shapedGlyphs(face, features)
		e.shapedSets[key] = set
	}
	return id, set != 0 && l.hbSetHas(set, id) == 0
}

// shapedGlyphs returns the set of glyphs of a face that the lookups of the
// features shaping applies with a feature list (Style.Features) take as
// input, in GSUB and GPOS: the glyphs they may match at, substitute or
// move.
func (e *pangoEngine) shapedGlyphs(face uintptr, list string) uintptr {
	l := &pangoLib
	on := map[string]bool{}
	for _, t := range hbDefaultFeatures {
		on[t] = true
	}
	for _, f := range features(list) {
		on[string(f.tag[:])] = f.value != 0
	}
	// The mark features attach marks to what comes before them: a text
	// without marks, which alone leaves out, never reaches them. The
	// others any value turns on.
	delete(on, "mark")
	delete(on, "mkmk")
	tags := make([]uint32, 0, len(on)+1)
	for t, ok := range on {
		if ok {
			tags = append(tags, uint32(t[0])<<24|uint32(t[1])<<16|uint32(t[2])<<8|uint32(t[3]))
		}
	}
	tags = append(tags, 0) // HB_TAG_NONE ends the list
	input, context := l.hbSetCreate(), l.hbSetCreate()
	defer l.hbSetDestroy(context)
	for _, table := range [...]uint32{tagGSUB, tagGPOS} {
		// All scripts and languages: Pango picks them by the text and
		// the locale.
		lookups := l.hbSetCreate()
		l.hbCollectLookups(face, table, nil, nil, &tags[0], lookups)
		lookup := ^uint32(0) // HB_SET_VALUE_INVALID starts the iteration
		for l.hbSetNext(lookups, &lookup) != 0 {
			l.hbLookupCollectGlyphs(face, table, lookup, context, input, context, context)
		}
		l.hbSetDestroy(lookups)
	}
	return input
}
