package pipeline

import (
	"testing"
)

func TestGetPriority(t *testing.T) {
	tests := []struct {
		r        rune
		expected int
	}{
		// High priority
		{'.', priorityHigh},
		{'!', priorityHigh},
		{'?', priorityHigh},
		{'\u3002', priorityHigh}, // 。
		{'\uff01', priorityHigh}, // ！
		{'\uff1f', priorityHigh}, // ？

		// Medium priority
		{';', priorityMedium},
		{':', priorityMedium},
		{')', priorityMedium},
		{']', priorityMedium},
		{'}', priorityMedium},
		{'\uff1b', priorityMedium}, // ；
		{'\uff1a', priorityMedium}, // ：
		{'\u300d', priorityMedium}, // 」

		// Low priority
		{',', priorityLow},
		{'(', priorityLow},
		{'[', priorityLow},
		{'-', priorityLow},
		{'\uff0c', priorityLow}, // ，
		{'\u3001', priorityLow}, // 、
		{'\u2026', priorityLow}, // …

		// None
		{'a', priorityNone},
		{'1', priorityNone},
		{' ', priorityNone},
	}

	for _, tt := range tests {
		got := getPriority(tt.r)
		if got != tt.expected {
			t.Errorf("getPriority(%q) = %d, want %d", tt.r, got, tt.expected)
		}
	}
}

func TestIsPunctuation(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'.', true},
		{',', true},
		{';', true},
		{'\u3002', true}, // 。
		{'\u2026', true}, // …
		{'a', false},
		{' ', false},
		{'0', false},
	}

	for _, tt := range tests {
		got := isPunctuation(tt.r)
		if got != tt.want {
			t.Errorf("isPunctuation(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
}

func TestWordEndsWithPunctuation(t *testing.T) {
	tests := []struct {
		text         string
		hasPunct     bool
		wantPriority int
	}{
		{"Hello.", true, priorityHigh},
		{"Hello!", true, priorityHigh},
		{"Hello?", true, priorityHigh},
		{"Hello,", true, priorityLow},
		{"Hello;", true, priorityMedium},
		{"Hello", false, priorityNone},
		{"", false, priorityNone},
		{"Hello. ", true, priorityHigh}, // trailing space should be trimmed
		{"\u3053\u3093\u306b\u3061\u306f\u3002", true, priorityHigh}, // こんにちは。
	}

	for _, tt := range tests {
		hasPunct, _, priority := wordEndsWithPunctuation(tt.text)
		if hasPunct != tt.hasPunct {
			t.Errorf("wordEndsWithPunctuation(%q): hasPunct = %v, want %v", tt.text, hasPunct, tt.hasPunct)
		}
		if priority != tt.wantPriority {
			t.Errorf("wordEndsWithPunctuation(%q): priority = %d, want %d", tt.text, priority, tt.wantPriority)
		}
	}
}

func TestFindSplitPosition(t *testing.T) {
	tests := []struct {
		text   string
		maxLen int
		want   int
	}{
		// Short text fits within maxLen.
		{"Hello", 10, 5},
		// Split at space — searches backwards from maxLen+1, finds space at index 11.
		{"Hello world foo", 11, 11},
		// Split at space — "Hello, world" maxLen=7, searches from index 7 back, finds space at 6.
		{"Hello, world", 7, 6},
		// No good split point, fall back to maxLen.
		{"Helloworldfoo", 5, 5},
	}

	for _, tt := range tests {
		got := findSplitPosition(tt.text, tt.maxLen)
		if got != tt.want {
			t.Errorf("findSplitPosition(%q, %d) = %d, want %d", tt.text, tt.maxLen, got, tt.want)
		}
	}
}

func TestEndsWithJoinPunctuation(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"Hello.", true},
		{"Hello,", true},
		{"Hello?", true},
		{"Hello", false},
		{"", false},
		{"\u3053\u3093\u306b\u3061\u306f\u3002", true}, // こんにちは。
	}

	for _, tt := range tests {
		got := endsWithJoinPunctuation(tt.text)
		if got != tt.want {
			t.Errorf("endsWithJoinPunctuation(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}
