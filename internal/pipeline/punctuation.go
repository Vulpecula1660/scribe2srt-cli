package pipeline

import (
	"strings"
	"unicode/utf8"
)

// Punctuation priority levels.
const (
	priorityHigh   = 0
	priorityMedium = 1
	priorityLow    = 2
	priorityNone   = -1
)

var highPriority = map[rune]struct{}{
	'.': {}, '!': {}, '?': {},
	'\u3002': {}, '\uff01': {}, '\uff1f': {}, // 。！？
}

var mediumPriority = map[rune]struct{}{
	';': {}, ':': {}, ')': {}, ']': {}, '}': {},
	'\uff1b': {}, '\uff1a': {}, '\u300b': {}, '\u300d': {}, '\u3011': {}, '\uff09': {}, // ；：》」】）
}

var lowPriority = map[rune]struct{}{
	',': {}, '(': {}, '[': {}, '{': {}, '-': {},
	'\uff0c': {}, '\u3001': {}, '\u300a': {}, '\u300c': {}, '\u3010': {}, '\uff08': {}, // ，、《「【（
}

// allPunctuation is the union of all priority sets.
var allPunctuation map[rune]struct{}

func init() {
	allPunctuation = make(map[rune]struct{}, len(highPriority)+len(mediumPriority)+len(lowPriority))
	for r := range highPriority {
		allPunctuation[r] = struct{}{}
	}
	for r := range mediumPriority {
		allPunctuation[r] = struct{}{}
	}
	for r := range lowPriority {
		allPunctuation[r] = struct{}{}
	}
	// Also add "..." and "…" — handled as individual chars in Python but
	// the ellipsis character is a single rune:
	allPunctuation['\u2026'] = struct{}{} // …
}

// getPriority returns the priority of a punctuation rune.
func getPriority(r rune) int {
	if _, ok := highPriority[r]; ok {
		return priorityHigh
	}
	if _, ok := mediumPriority[r]; ok {
		return priorityMedium
	}
	if _, ok := lowPriority[r]; ok {
		return priorityLow
	}
	if r == '\u2026' { // …
		return priorityLow
	}
	return priorityNone
}

// isPunctuation checks whether a rune is in any punctuation set.
func isPunctuation(r rune) bool {
	_, ok := allPunctuation[r]
	return ok
}

// wordEndsWithPunctuation checks if the text ends with a punctuation character.
// Returns (hasPunct, punctRune, priority).
func wordEndsWithPunctuation(text string) (bool, rune, int) {
	text = strings.TrimSpace(text)
	if text == "" {
		return false, 0, priorityNone
	}

	lastRune, _ := utf8.DecodeLastRuneInString(text)
	if lastRune == utf8.RuneError {
		return false, 0, priorityNone
	}

	priority := getPriority(lastRune)
	if priority >= 0 {
		return true, lastRune, priority
	}
	return false, 0, priorityNone
}

// findSplitPosition finds the best position to split text at or before maxLen (in runes).
// Returns a rune-index for the split point.
func findSplitPosition(text string, maxLen int) int {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return len(runes)
	}

	searchEnd := min(maxLen+1, len(runes))

	bestPos := -1
	for i := searchEnd - 1; i > 0; i-- {
		r := runes[i]
		if r == ' ' {
			bestPos = i
			break
		}
		if isPunctuation(r) {
			bestPos = i + 1
			break
		}
	}

	if bestPos <= 0 {
		bestPos = maxLen
	}
	return bestPos
}

// mergeJoinPunctuation contains punctuation characters that suppress space insertion
// when joining two subtitle texts. Matches the Python set in _merge_two_entries.
var mergeJoinPunctuation = map[rune]struct{}{
	'\u3002': {}, '\uff1f': {}, '\uff01': {}, '\u3001': {}, '\uff0c': {},
	'\uff1b': {}, '\uff1a': {},
	'\u201c': {}, '\u201d': {}, '\u2018': {}, '\u2019': {},
	'\uff08': {}, '\uff09': {}, '\u300a': {}, '\u300b': {},
	'\u300c': {}, '\u300d': {},
	'.': {}, '?': {}, '!': {}, ',': {}, ';': {}, ':': {},
	'(': {}, ')': {}, '"': {}, '\'': {}, '-': {},
}

// endsWithJoinPunctuation reports whether text ends with a punctuation character
// that suppresses space insertion when joining subtitles.
func endsWithJoinPunctuation(text string) bool {
	if text == "" {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(text)
	_, ok := mergeJoinPunctuation[r]
	return ok
}
