package tui

import "testing"

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
