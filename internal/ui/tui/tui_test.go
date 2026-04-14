package tui

import (
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestWrapText_TruncatesWithEllipsis(t *testing.T) {
	lines := wrapText("one two three", 6, 2)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[1] != "two..." {
		t.Errorf("expected truncated second line, got %q", lines[1])
	}
}

func TestWrapText_LongWordTruncates(t *testing.T) {
	lines := wrapText("superlongword", 5, 1)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if len(lines[0]) > 5 {
		t.Errorf("expected truncated line length <= 5, got %d", len(lines[0]))
	}
	if lines[0][len(lines[0])-1] != '.' {
		t.Errorf("expected ellipsis for truncated word, got %q", lines[0])
	}
}

func TestWrapText_ZeroWidth(t *testing.T) {
	lines := wrapText("text", 0, 2)
	if len(lines) != 1 || lines[0] != "" {
		t.Errorf("expected empty line for zero width, got %v", lines)
	}
}

func TestPadLines(t *testing.T) {
	lines := padLines([]string{"a"}, 3)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[1] != "" || lines[2] != "" {
		t.Errorf("expected padded lines to be empty, got %v", lines)
	}
}

func TestTruncateTableValue_UsesVisualWidth(t *testing.T) {
	value := "你你你你"
	got := truncateTableValue(value, 5)
	if runewidth.StringWidth(got) > 5 {
		t.Fatalf("expected visual width <= 5, got %d for %q", runewidth.StringWidth(got), got)
	}
	if got != "你..." {
		t.Errorf("expected ellipsis truncation to preserve whole runes, got %q", got)
	}
}

func TestTruncateTableValueFromStart_PreservesTail(t *testing.T) {
	value := "abcdefghijklmno"
	got := truncateTableValueFromStart(value, 10)
	if runewidth.StringWidth(got) > 10 {
		t.Fatalf("expected visual width <= 10, got %d for %q", runewidth.StringWidth(got), got)
	}
	if len(got) < 3 || got[:3] != "..." {
		t.Fatalf("expected leading ellipsis, got %q", got)
	}
}
