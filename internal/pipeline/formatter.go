package pipeline

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// formatSRTTime converts seconds to SRT time format HH:MM:SS,mmm.
func formatSRTTime(seconds float64) string {
	totalSec := math.Abs(seconds)
	hours := int(totalSec / 3600)
	remainder := math.Mod(totalSec, 3600)
	minutes := int(remainder / 60)
	secs := math.Mod(remainder, 60)
	millis := int(math.Mod(secs, 1) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, int(secs), millis)
}

// optimizeTextDisplay returns text on a single line if it fits within maxCPL,
// otherwise splits it into at most two lines.
func optimizeTextDisplay(text string, maxCPL int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	if utf8.RuneCountInString(text) <= maxCPL {
		return text
	}
	return splitTextIntoLines(text, maxCPL)
}

// splitTextIntoLines splits text into a maximum of two lines using
// findSplitPosition for intelligent break points.
func splitTextIntoLines(text string, maxCPL int) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= maxCPL {
		return text
	}

	splitPos := findSplitPosition(text, maxCPL)

	firstLine := strings.TrimSpace(string(runes[:splitPos]))
	remaining := strings.TrimSpace(string(runes[splitPos:]))

	if remaining == "" {
		return firstLine
	}
	return firstLine + "\n" + remaining
}
