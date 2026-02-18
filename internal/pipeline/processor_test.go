package pipeline

import (
	"strings"
	"testing"

	"scribe2srt/internal/config"
)

func defaultSettings() *config.SubtitleSettings {
	return &config.SubtitleSettings{
		MinSubtitleDuration: 0.83,
		MaxSubtitleDuration: 12.0,
		MinSubtitleGap:      0.083,
		CJKCPS:              11,
		LatinCPS:            15,
		CJKCharsPerLine:     25,
		LatinCharsPerLine:   42,
	}
}

func TestProcess_Empty(t *testing.T) {
	transcript := &TranscriptResponse{
		LanguageCode: "en",
		Words:        nil,
	}
	result := Process(transcript, defaultSettings())
	if result != "" {
		t.Errorf("expected empty result for empty transcript, got %q", result)
	}
}

func TestProcess_SingleSentence(t *testing.T) {
	transcript := &TranscriptResponse{
		LanguageCode: "en",
		Text:         "Hello world.",
		Words: []Word{
			{Text: "Hello ", Start: 0, End: 0.5, Type: "word"},
			{Text: "world.", Start: 0.5, End: 1.0, Type: "word"},
		},
	}

	result := Process(transcript, defaultSettings())
	if result == "" {
		t.Fatal("expected non-empty SRT output")
	}
	if !strings.Contains(result, "Hello world.") {
		t.Errorf("SRT output should contain 'Hello world.', got:\n%s", result)
	}
	if !strings.HasPrefix(result, "1\n") {
		t.Errorf("SRT should start with sequence number 1, got:\n%s", result)
	}
	if !strings.Contains(result, "-->") {
		t.Errorf("SRT should contain timing arrow, got:\n%s", result)
	}
}

func TestProcess_MultipleSentences(t *testing.T) {
	transcript := &TranscriptResponse{
		LanguageCode: "en",
		Text:         "Hello. Goodbye.",
		Words: []Word{
			{Text: "Hello.", Start: 0, End: 1, Type: "word"},
			{Text: " ", Start: 1, End: 1, Type: "spacing"},
			{Text: "Goodbye.", Start: 2, End: 3, Type: "word"},
		},
	}

	result := Process(transcript, defaultSettings())

	// Should have at least the sentence text.
	if !strings.Contains(result, "Hello.") {
		t.Errorf("SRT should contain 'Hello.', got:\n%s", result)
	}
	if !strings.Contains(result, "Goodbye.") {
		t.Errorf("SRT should contain 'Goodbye.', got:\n%s", result)
	}
}

func TestProcess_WithAudioEvents(t *testing.T) {
	transcript := &TranscriptResponse{
		LanguageCode: "en",
		Text:         "Hello.",
		Words: []Word{
			{Text: "Hello.", Start: 0, End: 1, Type: "word"},
			{Text: "(laughter)", Start: 1.5, End: 2, Type: "audio_event"},
		},
	}

	result := Process(transcript, defaultSettings())
	if !strings.Contains(result, "Hello.") {
		t.Errorf("SRT should contain 'Hello.', got:\n%s", result)
	}
	if !strings.Contains(result, "(laughter)") {
		t.Errorf("SRT should contain '(laughter)', got:\n%s", result)
	}
}

func TestProcess_CJK(t *testing.T) {
	transcript := &TranscriptResponse{
		LanguageCode: "ja",
		Text:         "\u3053\u3093\u306b\u3061\u306f\u3002",
		Words: []Word{
			{Text: "\u3053\u3093\u306b\u3061\u306f", Start: 0, End: 1, Type: "word"},
			{Text: "\u3002", Start: 1, End: 1.1, Type: "word"}, // 。standalone
		},
	}

	result := Process(transcript, defaultSettings())
	if result == "" {
		t.Fatal("expected non-empty SRT output for CJK")
	}
	// The 。should have been merged into the previous word during preprocessing.
	if !strings.Contains(result, "\u3053\u3093\u306b\u3061\u306f\u3002") {
		t.Errorf("SRT should contain merged CJK text, got:\n%s", result)
	}
}

func TestProcess_SRTFormat(t *testing.T) {
	transcript := &TranscriptResponse{
		LanguageCode: "en",
		Text:         "First. Second.",
		Words: []Word{
			{Text: "First.", Start: 0, End: 1, Type: "word"},
			{Text: " ", Start: 1, End: 1, Type: "spacing"},
			{Text: "Second.", Start: 3, End: 4, Type: "word"},
		},
	}

	result := Process(transcript, defaultSettings())
	lines := strings.Split(result, "\n")

	// SRT format: number, timing, text, blank line, ...
	// First entry should start with "1".
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines in SRT, got %d", len(lines))
	}
	if lines[0] != "1" {
		t.Errorf("first line should be '1', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "-->") {
		t.Errorf("second line should contain '-->', got %q", lines[1])
	}
}

func TestProcess_OnlyAudioEvents(t *testing.T) {
	transcript := &TranscriptResponse{
		LanguageCode: "en",
		Words: []Word{
			{Text: "(music)", Start: 0, End: 2, Type: "audio_event"},
			{Text: "(applause)", Start: 3, End: 5, Type: "audio_event"},
		},
	}

	result := Process(transcript, defaultSettings())
	if !strings.Contains(result, "(music)") {
		t.Errorf("SRT should contain '(music)', got:\n%s", result)
	}
	if !strings.Contains(result, "(applause)") {
		t.Errorf("SRT should contain '(applause)', got:\n%s", result)
	}
}

func TestGenerateSRT_Empty(t *testing.T) {
	result := generateSRT(nil, 42)
	if result != "" {
		t.Errorf("expected empty string for nil entries, got %q", result)
	}
}

func TestCreateAudioEventEntries(t *testing.T) {
	events := []Word{
		{Text: "(music)", Start: 0, End: 2, Type: "audio_event"},
	}

	entries := createAudioEventEntries(events)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].IsAudioEvent {
		t.Error("expected IsAudioEvent=true")
	}
	if entries[0].Text != "(music)" {
		t.Errorf("Text = %q, want '(music)'", entries[0].Text)
	}
}
