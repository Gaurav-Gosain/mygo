package ui

import (
	"math/rand/v2"
	"os"
	"testing"
)

// TestShapeAlone checks that text AppendShapeAlone lays out takes the
// glyphs and advances Shape gives it, in a coding font with ligatures on
// and off, and that it declines text with ligatures.
func TestShapeAlone(t *testing.T) {
	path := os.Getenv("MYGO_TEST_MONO_FONT")
	if path == "" {
		t.Skip("MYGO_TEST_MONO_FONT names no font file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := RegisterFont(data, "Alone Test Mono"); err != nil {
		t.Fatal(err)
	}
	alphabet := []rune(" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~éüñßÆø")
	rng := rand.New(rand.NewPCG(1, 2))
	for _, features := range []string{"", "calt=0,liga=0,dlig=0"} {
		for _, weight := range []int{400, 700} {
			f := Font{Family: "Alone Test Mono", Size: 13, Weight: weight, Features: features}
			fast := 0
			for range 20000 {
				n := 1 + rng.IntN(12)
				text := make([]rune, n)
				for i := range text {
					text[i] = alphabet[rng.IntN(len(alphabet))]
				}
				got, ok := AppendShapeAlone(nil, text, f)
				if !ok {
					continue
				}
				fast++
				want := Shape(string(text), f)
				if len(got) != len(want) {
					t.Fatalf("%q (%q): %d glyphs alone, Shape %d", string(text), features, len(got), len(want))
				}
				for i := range got {
					if got[i] != want[i] {
						t.Fatalf("%q (%q): glyph %d alone %+v, Shape %+v", string(text), features, i, got[i], want[i])
					}
				}
			}
			t.Logf("features %q weight %d: %d of 20000 laid out alone", features, weight, fast)
		}
	}
	// Ligatures and contextual alternates need shaping.
	for _, s := range []string{"->", "!=", "==", "www", "0xF", "fi"} {
		f := Font{Family: "Alone Test Mono", Size: 13}
		if got, ok := AppendShapeAlone(nil, []rune(s), f); ok {
			want := Shape(s, f)
			if len(got) != len(want) {
				t.Errorf("%q laid out alone, but shapes to %d glyphs", s, len(want))
			}
		}
	}
}
