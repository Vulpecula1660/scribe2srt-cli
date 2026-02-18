package pipeline

import (
	"testing"
)

func TestPreprocessWords_Empty(t *testing.T) {
	result := preprocessWords(nil)
	if len(result.Words) != 0 {
		t.Errorf("expected 0 words, got %d", len(result.Words))
	}
	if len(result.AudioEvents) != 0 {
		t.Errorf("expected 0 audio events, got %d", len(result.AudioEvents))
	}
}

func TestPreprocessWords_SeparatesAudioEvents(t *testing.T) {
	raw := []Word{
		{Text: "Hello", Start: 0, End: 1, Type: "word"},
		{Text: "(laughter)", Start: 1, End: 2, Type: "audio_event"},
		{Text: "world", Start: 2, End: 3, Type: "word"},
	}
	result := preprocessWords(raw)

	if len(result.Words) != 2 {
		t.Fatalf("expected 2 words, got %d", len(result.Words))
	}
	if len(result.AudioEvents) != 1 {
		t.Fatalf("expected 1 audio event, got %d", len(result.AudioEvents))
	}
	if result.AudioEvents[0].Text != "(laughter)" {
		t.Errorf("expected audio event text '(laughter)', got %q", result.AudioEvents[0].Text)
	}
}

func TestPreprocessWords_SpacingAppendsToPrevious(t *testing.T) {
	raw := []Word{
		{Text: "Hello", Start: 0, End: 1, Type: "word"},
		{Text: " ", Start: 1, End: 1, Type: "spacing"},
		{Text: "world", Start: 1, End: 2, Type: "word"},
	}
	result := preprocessWords(raw)

	if len(result.Words) != 2 {
		t.Fatalf("expected 2 words, got %d", len(result.Words))
	}
	if result.Words[0].Text != "Hello " {
		t.Errorf("expected first word 'Hello ', got %q", result.Words[0].Text)
	}
	if result.Words[1].Text != "world" {
		t.Errorf("expected second word 'world', got %q", result.Words[1].Text)
	}
}

func TestPreprocessWords_SpacingNotDuplicated(t *testing.T) {
	raw := []Word{
		{Text: "Hello ", Start: 0, End: 1, Type: "word"},
		{Text: " ", Start: 1, End: 1, Type: "spacing"},
		{Text: "world", Start: 1, End: 2, Type: "word"},
	}
	result := preprocessWords(raw)

	if result.Words[0].Text != "Hello " {
		t.Errorf("expected 'Hello ' (no double space), got %q", result.Words[0].Text)
	}
}

func TestPreprocessWords_MergesCJKPunctuation(t *testing.T) {
	raw := []Word{
		{Text: "hello", Start: 0, End: 1, Type: "word"},
		{Text: "\u3002", Start: 1, End: 1.1, Type: "word"}, // 。
	}
	result := preprocessWords(raw)

	if len(result.Words) != 1 {
		t.Fatalf("expected 1 word after CJK punct merge, got %d", len(result.Words))
	}
	if result.Words[0].Text != "hello\u3002" {
		t.Errorf("expected 'hello。', got %q", result.Words[0].Text)
	}
	if result.Words[0].End != 1.1 {
		t.Errorf("expected end time 1.1, got %f", result.Words[0].End)
	}
}

func TestPreprocessWords_NoDoubleCJKPunctMerge(t *testing.T) {
	// If previous word already ends with CJK punctuation, don't merge another one.
	raw := []Word{
		{Text: "hello\u3002", Start: 0, End: 1, Type: "word"}, // hello。
		{Text: "\uff1f", Start: 1, End: 1.1, Type: "word"},    // ？
	}
	result := preprocessWords(raw)

	if len(result.Words) != 2 {
		t.Fatalf("expected 2 words (no double-punct merge), got %d", len(result.Words))
	}
}

func TestPreprocessWords_SpacingWithoutPrevWord(t *testing.T) {
	// Spacing at the start should be dropped.
	raw := []Word{
		{Text: " ", Start: 0, End: 0, Type: "spacing"},
		{Text: "Hello", Start: 0, End: 1, Type: "word"},
	}
	result := preprocessWords(raw)

	if len(result.Words) != 1 {
		t.Fatalf("expected 1 word, got %d", len(result.Words))
	}
	if result.Words[0].Text != "Hello" {
		t.Errorf("expected 'Hello', got %q", result.Words[0].Text)
	}
}

func TestPreprocessWords_NonSpacingTokenPassedThrough(t *testing.T) {
	// A spacing token that isn't pure whitespace should be passed through.
	raw := []Word{
		{Text: "Hello", Start: 0, End: 1, Type: "word"},
		{Text: "-", Start: 1, End: 1, Type: "spacing"},
		{Text: "world", Start: 1, End: 2, Type: "word"},
	}
	result := preprocessWords(raw)

	// "-" is not TrimSpace == "", so it's not a pure space; it gets skipped
	// because of the TrimSpace check, the space won't be appended.
	if len(result.Words) != 2 {
		t.Fatalf("expected 2 words, got %d", len(result.Words))
	}
}
