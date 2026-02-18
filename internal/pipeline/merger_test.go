package pipeline

import (
	"math"
	"testing"

	"scribe2srt/internal/config"
)

func defaultMerger() *IntelligentMerger {
	settings := &config.SubtitleSettings{
		MinSubtitleDuration: 0.83,
		MaxSubtitleDuration: 12.0,
		MinSubtitleGap:      0.083,
		CJKCPS:              11,
		LatinCPS:            15,
		CJKCharsPerLine:     25,
		LatinCharsPerLine:   42,
	}
	return NewIntelligentMerger("en", settings)
}

func cjkMerger() *IntelligentMerger {
	settings := &config.SubtitleSettings{
		MinSubtitleDuration: 0.83,
		MaxSubtitleDuration: 12.0,
		MinSubtitleGap:      0.083,
		CJKCPS:              11,
		LatinCPS:            15,
		CJKCharsPerLine:     25,
		LatinCharsPerLine:   42,
	}
	return NewIntelligentMerger("ja", settings)
}

func TestNewIntelligentMerger_Latin(t *testing.T) {
	m := defaultMerger()
	if m.IsCJK {
		t.Error("expected IsCJK=false for 'en'")
	}
	if m.MaxCPS != 15 {
		t.Errorf("MaxCPS = %f, want 15", m.MaxCPS)
	}
	if m.MaxCharsPerLine != 42 {
		t.Errorf("MaxCharsPerLine = %d, want 42", m.MaxCharsPerLine)
	}
}

func TestNewIntelligentMerger_CJK(t *testing.T) {
	m := cjkMerger()
	if !m.IsCJK {
		t.Error("expected IsCJK=true for 'ja'")
	}
	if m.MaxCPS != 11 {
		t.Errorf("MaxCPS = %f, want 11", m.MaxCPS)
	}
	if m.MaxCharsPerLine != 25 {
		t.Errorf("MaxCharsPerLine = %d, want 25", m.MaxCharsPerLine)
	}
}

func TestStripWhitespaceCount(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"hello", 5},
		{"hello world", 10},
		{"  spaces  ", 6},
		{"", 0},
		{"\t\n", 0},
	}
	for _, tt := range tests {
		got := stripWhitespaceCount(tt.text)
		if got != tt.want {
			t.Errorf("stripWhitespaceCount(%q) = %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestCalculateCPS(t *testing.T) {
	m := defaultMerger()

	// Normal case.
	cps := m.calculateCPS("hello", 1.0)
	if cps != 5.0 {
		t.Errorf("calculateCPS('hello', 1.0) = %f, want 5.0", cps)
	}

	// Zero duration → +Inf.
	cps = m.calculateCPS("hello", 0.0)
	if !math.IsInf(cps, 1) {
		t.Errorf("calculateCPS('hello', 0.0) = %f, want +Inf", cps)
	}

	// Text with spaces — only non-whitespace counted.
	cps = m.calculateCPS("hello world", 1.0)
	if cps != 10.0 {
		t.Errorf("calculateCPS('hello world', 1.0) = %f, want 10.0", cps)
	}
}

func TestGetDynamicCPSLimit(t *testing.T) {
	m := defaultMerger() // MaxCPS = 15

	tests := []struct {
		text string
		want float64
	}{
		{"ab", 45.0},    // <= 3 chars → 3x
		{"abc", 45.0},   // <= 3 chars → 3x
		{"abcd", 30.0},  // <= 5 chars → 2x
		{"abcde", 30.0}, // <= 5 chars → 2x
		{"abcdefghij", 22.5},       // <= 10 chars → 1.5x
		{"abcdefghijk", 15.0},      // > 10 chars → 1x
		{"a long sentence here", 15.0}, // > 10 non-space chars → 1x
	}

	for _, tt := range tests {
		got := m.getDynamicCPSLimit(tt.text)
		if got != tt.want {
			t.Errorf("getDynamicCPSLimit(%q) = %f, want %f", tt.text, got, tt.want)
		}
	}
}

func TestCalculateDisplayLines(t *testing.T) {
	m := defaultMerger() // MaxCharsPerLine = 42

	tests := []struct {
		text string
		want int
	}{
		{"", 0},
		{"Short text", 1},
		{"This is a longer piece of text that exceeds forty two characters and needs wrapping", 3},
	}

	for _, tt := range tests {
		got := m.calculateDisplayLines(tt.text)
		if got != tt.want {
			t.Errorf("calculateDisplayLines(%q) = %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestCanMerge_AudioEvents(t *testing.T) {
	m := defaultMerger()

	e1 := SubtitleEntry{Text: "Hello", Start: 0, End: 1, IsAudioEvent: true}
	e2 := SubtitleEntry{Text: "World", Start: 1.1, End: 2}

	can, reason := m.canMerge(e1, e2)
	if can {
		t.Error("should not merge audio events")
	}
	if reason != "audio event" {
		t.Errorf("reason = %q, want 'audio event'", reason)
	}
}

func TestCanMerge_GapTooLarge(t *testing.T) {
	m := defaultMerger()

	e1 := SubtitleEntry{Text: "Hello", Start: 0, End: 1}
	e2 := SubtitleEntry{Text: "World", Start: 3.5, End: 4.5}

	can, reason := m.canMerge(e1, e2)
	if can {
		t.Error("should not merge when gap > 2.0s")
	}
	if reason != "gap too large" {
		t.Errorf("reason = %q, want 'gap too large'", reason)
	}
}

func TestCanMerge_GapTooSmall(t *testing.T) {
	m := defaultMerger()

	e1 := SubtitleEntry{Text: "Hello", Start: 0, End: 1}
	e2 := SubtitleEntry{Text: "World", Start: 1.01, End: 2}

	can, reason := m.canMerge(e1, e2)
	if can {
		t.Error("should not merge when gap < MinSubtitleGap")
	}
	if reason != "gap too small" {
		t.Errorf("reason = %q, want 'gap too small'", reason)
	}
}

func TestCanMerge_DurationTooLong(t *testing.T) {
	m := defaultMerger()

	e1 := SubtitleEntry{Text: "Hello", Start: 0, End: 3}
	e2 := SubtitleEntry{Text: "World", Start: 3.1, End: 6.5}

	can, reason := m.canMerge(e1, e2)
	if can {
		t.Error("should not merge when merged duration > 6.0s")
	}
	if reason != "duration too long" {
		t.Errorf("reason = %q, want 'duration too long'", reason)
	}
}

func TestCanMerge_Success(t *testing.T) {
	m := defaultMerger()

	e1 := SubtitleEntry{Text: "Hello there", Start: 0, End: 1}
	e2 := SubtitleEntry{Text: "my friend", Start: 1.1, End: 2}

	can, reason := m.canMerge(e1, e2)
	if !can {
		t.Errorf("expected merge to succeed, got reason: %q", reason)
	}
}

func TestCalculateMergeBenefit(t *testing.T) {
	m := defaultMerger()

	// Two very short entries with small gap → high benefit.
	e1 := SubtitleEntry{Text: "Hi", Start: 0, End: 0.3, CharCount: 2}
	e2 := SubtitleEntry{Text: "there", Start: 0.4, End: 0.7, CharCount: 5}

	benefit := m.calculateMergeBenefit(e1, e2)
	if benefit <= 5.0 {
		t.Errorf("expected high benefit for short entries with small gap, got %f", benefit)
	}

	// Two long entries with large gap → low/no benefit.
	e3 := SubtitleEntry{Text: "A long subtitle entry with many words", Start: 0, End: 3, CharCount: 33}
	e4 := SubtitleEntry{Text: "Another long subtitle entry here", Start: 4, End: 7, CharCount: 28}

	benefit2 := m.calculateMergeBenefit(e3, e4)
	if benefit2 >= benefit {
		t.Errorf("expected lower benefit for long entries, got %f >= %f", benefit2, benefit)
	}
}

func TestMergeTwoEntries_Latin(t *testing.T) {
	m := defaultMerger()

	e1 := SubtitleEntry{
		Text:  "Hello",
		Start: 0, End: 1,
		Words:     []Word{{Text: "Hello", Start: 0, End: 1, Type: "word"}},
		WordCount: 1, CharCount: 5,
	}
	e2 := SubtitleEntry{
		Text:  "world",
		Start: 1.1, End: 2,
		Words:     []Word{{Text: "world", Start: 1.1, End: 2, Type: "word"}},
		WordCount: 1, CharCount: 5,
	}

	merged := m.mergeTwoEntries(e1, e2)
	if merged.Text != "Hello world" {
		t.Errorf("merged.Text = %q, want 'Hello world'", merged.Text)
	}
	if merged.Start != 0 || merged.End != 2 {
		t.Errorf("merged timing = [%f, %f], want [0, 2]", merged.Start, merged.End)
	}
	if merged.WordCount != 2 {
		t.Errorf("merged.WordCount = %d, want 2", merged.WordCount)
	}
	if len(merged.Words) != 2 {
		t.Errorf("len(merged.Words) = %d, want 2", len(merged.Words))
	}
}

func TestMergeTwoEntries_CJK(t *testing.T) {
	m := cjkMerger()

	e1 := SubtitleEntry{
		Text:  "\u3053\u3093\u306b\u3061\u306f", // こんにちは
		Start: 0, End: 1,
		Words:     []Word{{Text: "\u3053\u3093\u306b\u3061\u306f", Start: 0, End: 1, Type: "word"}},
		WordCount: 1, CharCount: 5,
	}
	e2 := SubtitleEntry{
		Text:  "\u4e16\u754c", // 世界
		Start: 1.1, End: 2,
		Words:     []Word{{Text: "\u4e16\u754c", Start: 1.1, End: 2, Type: "word"}},
		WordCount: 1, CharCount: 2,
	}

	merged := m.mergeTwoEntries(e1, e2)
	// CJK should not add space.
	if merged.Text != "\u3053\u3093\u306b\u3061\u306f\u4e16\u754c" {
		t.Errorf("merged.Text = %q, want no space between CJK texts", merged.Text)
	}
}

func TestMergeTwoEntries_JoinPunctuation(t *testing.T) {
	m := defaultMerger()

	e1 := SubtitleEntry{
		Text:      "Hello,",
		Start:     0, End: 1,
		Words:     []Word{{Text: "Hello,", Start: 0, End: 1, Type: "word"}},
		WordCount: 1, CharCount: 6,
	}
	e2 := SubtitleEntry{
		Text:      "world",
		Start:     1.1, End: 2,
		Words:     []Word{{Text: "world", Start: 1.1, End: 2, Type: "word"}},
		WordCount: 1, CharCount: 5,
	}

	merged := m.mergeTwoEntries(e1, e2)
	// Ends with join punctuation → no space.
	if merged.Text != "Hello,world" {
		t.Errorf("merged.Text = %q, want 'Hello,world'", merged.Text)
	}
}

func TestMergeBasicEntries_Empty(t *testing.T) {
	m := defaultMerger()
	result := m.MergeBasicEntries(nil)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestMergeBasicEntries_MergesShortEntries(t *testing.T) {
	m := defaultMerger()

	entries := []SubtitleEntry{
		{Text: "Hi", Start: 0, End: 0.3, CharCount: 2, WordCount: 1,
			Words: []Word{{Text: "Hi", Start: 0, End: 0.3, Type: "word"}}},
		{Text: "there", Start: 0.4, End: 0.7, CharCount: 5, WordCount: 1,
			Words: []Word{{Text: "there", Start: 0.4, End: 0.7, Type: "word"}}},
	}

	merged := m.MergeBasicEntries(entries)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged entry, got %d", len(merged))
	}
}

func TestMergeBasicEntries_DoesNotMergeDistantEntries(t *testing.T) {
	m := defaultMerger()

	entries := []SubtitleEntry{
		{Text: "Hello there my friend", Start: 0, End: 2, CharCount: 18, WordCount: 4,
			Words: []Word{{Text: "Hello", Start: 0, End: 2, Type: "word"}}},
		{Text: "Goodbye my dear", Start: 5, End: 7, CharCount: 13, WordCount: 3,
			Words: []Word{{Text: "Goodbye", Start: 5, End: 7, Type: "word"}}},
	}

	merged := m.MergeBasicEntries(entries)
	if len(merged) != 2 {
		t.Fatalf("expected 2 entries (gap too large), got %d", len(merged))
	}
}

func TestOptimizeMergedEntries_EnforcesMinDuration(t *testing.T) {
	m := defaultMerger()

	entries := []SubtitleEntry{
		{Text: "Hi", Start: 0, End: 0.1}, // duration 0.1 < min 0.83
	}

	optimized := m.OptimizeMergedEntries(entries)
	if len(optimized) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(optimized))
	}

	duration := optimized[0].End - optimized[0].Start
	if duration < m.MinSubtitleDuration-0.001 {
		t.Errorf("duration = %f, want >= %f", duration, m.MinSubtitleDuration)
	}
}

func TestOptimizeMergedEntries_EnforcesMaxDuration(t *testing.T) {
	m := defaultMerger()

	entries := []SubtitleEntry{
		{Text: "Very long subtitle", Start: 0, End: 15}, // duration 15 > max 12
	}

	optimized := m.OptimizeMergedEntries(entries)
	duration := optimized[0].End - optimized[0].Start
	if duration > m.MaxSubtitleDuration+0.001 {
		t.Errorf("duration = %f, want <= %f", duration, m.MaxSubtitleDuration)
	}
}

func TestOptimizeMergedEntries_EnforcesMinGap(t *testing.T) {
	m := defaultMerger()

	entries := []SubtitleEntry{
		{Text: "First", Start: 0, End: 1.0},
		{Text: "Second", Start: 1.01, End: 2.0}, // gap 0.01 < min 0.083
	}

	optimized := m.OptimizeMergedEntries(entries)
	gap := optimized[1].Start - optimized[0].End
	if gap < m.MinSubtitleGap-0.001 {
		t.Errorf("gap = %f, want >= %f", gap, m.MinSubtitleGap)
	}
}

func TestOptimizeMergedEntries_Empty(t *testing.T) {
	m := defaultMerger()
	result := m.OptimizeMergedEntries(nil)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}
