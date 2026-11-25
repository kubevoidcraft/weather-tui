package ui

import (
	"testing"
	"unicode/utf8"
)

func TestSparklineLength(t *testing.T) {
	in := []float64{1, 2, 3, 4, 5, 6}
	out := sparkline(in)
	if utf8.RuneCountInString(out) != len(in) {
		t.Fatalf("expected %d runes, got %d (%q)", len(in), utf8.RuneCountInString(out), out)
	}
}

func TestSparklineFlatSeries(t *testing.T) {
	out := sparkline([]float64{5, 5, 5, 5})
	// Flat series should still produce 4 non-empty runes.
	if utf8.RuneCountInString(out) != 4 {
		t.Fatalf("expected 4 runes for flat series, got %q", out)
	}
}

func TestSparklineMonotonic(t *testing.T) {
	out := []rune(sparkline([]float64{1, 2, 3, 4, 5, 6, 7, 8}))
	for i := 1; i < len(out); i++ {
		if out[i] < out[i-1] {
			t.Fatalf("expected monotonic non-decreasing blocks, got %q", string(out))
		}
	}
}

func TestSparklineEmpty(t *testing.T) {
	if sparkline(nil) != "" {
		t.Fatalf("expected empty string for nil input")
	}
}

func TestShadedClamping(t *testing.T) {
	out := shaded([]float64{-10, 0, 50, 100, 500})
	if utf8.RuneCountInString(out) != 5 {
		t.Fatalf("expected 5 runes, got %q", out)
	}
}

// Low probabilities should render as the "empty frame" glyph rather than a
// shaded block that falsely suggests rain is possible.
func TestShadedLowProbabilityIsHollow(t *testing.T) {
	out := []rune(shaded([]float64{0, 5, 9}))
	for i, r := range out {
		if r != shadeBlocks[0] {
			t.Fatalf("expected hollow frame at position %d, got %q", i, string(r))
		}
	}
}

// At the first non-zero threshold the cell should become visibly shaded.
func TestShadedThresholdCrossings(t *testing.T) {
	out := []rune(shaded([]float64{10, 30, 60, 85}))
	expected := []rune{'░', '▒', '▓', '█'}
	for i, want := range expected {
		if out[i] != want {
			t.Fatalf("index %d: want %q, got %q", i, string(want), string(out[i]))
		}
	}
}

func TestShadedMonotonic(t *testing.T) {
	// Runes in shadeBlocks are not codepoint-ordered (█ is U+2588 but comes
	// last visually), so compare indices within shadeBlocks instead.
	indexOf := func(r rune) int {
		for i, b := range shadeBlocks {
			if b == r {
				return i
			}
		}
		return -1
	}

	out := []rune(shaded([]float64{0, 25, 50, 75, 100}))
	for i := 1; i < len(out); i++ {
		if indexOf(out[i]) < indexOf(out[i-1]) {
			t.Fatalf("expected non-decreasing shade index, got %q", string(out))
		}
	}
}
