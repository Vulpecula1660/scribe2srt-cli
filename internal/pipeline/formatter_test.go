package pipeline

import (
	"testing"
)

func TestFormatSRTTime(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{0, "00:00:00,000"},
		{1.5, "00:00:01,500"},
		{61.123, "00:01:01,122"},  // floating point: math.Mod(61.123, 1)*1000 ≈ 122
		{3661.999, "01:01:01,998"}, // floating point: math.Mod(1.999, 1)*1000 ≈ 998
		{3600, "01:00:00,000"},
		{0.083, "00:00:00,083"},
		{7200.5, "02:00:00,500"},
	}

	for _, tt := range tests {
		got := formatSRTTime(tt.seconds)
		if got != tt.want {
			t.Errorf("formatSRTTime(%f) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestOptimizeTextDisplay_ShortText(t *testing.T) {
	// Text shorter than maxCPL should be returned as-is.
	result := optimizeTextDisplay("Hello world", 42)
	if result != "Hello world" {
		t.Errorf("got %q, want 'Hello world'", result)
	}
}

func TestOptimizeTextDisplay_Empty(t *testing.T) {
	result := optimizeTextDisplay("", 42)
	if result != "" {
		t.Errorf("got %q, want empty string", result)
	}
}

func TestOptimizeTextDisplay_LongText(t *testing.T) {
	text := "This is a very long subtitle text that definitely exceeds the maximum characters per line limit"
	result := optimizeTextDisplay(text, 42)

	// Should contain a newline for line splitting.
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	// The result should have been split.
	lines := 0
	for _, c := range result {
		if c == '\n' {
			lines++
		}
	}
	if lines != 1 {
		t.Errorf("expected exactly 1 newline (2 lines), got %d newlines", lines)
	}
}

func TestSplitTextIntoLines_ShortText(t *testing.T) {
	result := splitTextIntoLines("Hello", 42)
	if result != "Hello" {
		t.Errorf("got %q, want 'Hello'", result)
	}
}

func TestSplitTextIntoLines_SplitsAtSpace(t *testing.T) {
	// "Hello world foo bar baz" with maxCPL=12 should split around position 12.
	text := "Hello world foo bar baz"
	result := splitTextIntoLines(text, 12)

	if result == text {
		t.Error("expected text to be split, but got original")
	}
	// Verify it contains a newline.
	found := false
	for _, c := range result {
		if c == '\n' {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected newline in result %q", result)
	}
}
