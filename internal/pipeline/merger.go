package pipeline

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"scribe2srt/internal/config"
)

// IntelligentMerger implements Stage 2 of the subtitle pipeline.
type IntelligentMerger struct {
	Language           string
	IsCJK              bool
	MinSubtitleDuration float64
	MaxSubtitleDuration float64
	MinSubtitleGap     float64
	MaxCPS             float64
	MaxCharsPerLine    int
}

// NewIntelligentMerger creates a merger from subtitle settings and language code.
func NewIntelligentMerger(langCode string, settings *config.SubtitleSettings) *IntelligentMerger {
	lang := langCode
	if len(lang) > 3 {
		lang = lang[:3]
	}
	isCJK := config.IsCJK(lang)

	m := &IntelligentMerger{
		Language:           lang,
		IsCJK:              isCJK,
		MinSubtitleDuration: settings.MinSubtitleDuration,
		MaxSubtitleDuration: settings.MaxSubtitleDuration,
		MinSubtitleGap:     settings.MinSubtitleGap,
	}

	if isCJK {
		m.MaxCPS = settings.CJKCPS
		m.MaxCharsPerLine = settings.CJKCharsPerLine
	} else {
		m.MaxCPS = settings.LatinCPS
		m.MaxCharsPerLine = settings.LatinCharsPerLine
	}

	return m
}

// stripWhitespaceCount returns the number of non-whitespace runes in text.
func stripWhitespaceCount(text string) int {
	count := 0
	for _, r := range text {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

func (m *IntelligentMerger) calculateCPS(text string, duration float64) float64 {
	if duration <= 0 {
		return math.Inf(1)
	}
	return float64(stripWhitespaceCount(text)) / duration
}

func (m *IntelligentMerger) getDynamicCPSLimit(text string) float64 {
	base := m.MaxCPS
	textLen := stripWhitespaceCount(text)

	if textLen <= 3 {
		return base * 3.0
	} else if textLen <= 5 {
		return base * 2.0
	} else if textLen <= 10 {
		return base * 1.5
	}
	return base
}

func (m *IntelligentMerger) calculateDisplayLines(text string) int {
	if text == "" {
		return 0
	}

	remaining := strings.TrimSpace(text)
	lines := 0

	for remaining != "" {
		lines++
		runes := []rune(remaining)
		if len(runes) <= m.MaxCharsPerLine {
			break
		}
		splitPos := findSplitPosition(remaining, m.MaxCharsPerLine)
		splitRunes := []rune(remaining)
		remaining = strings.TrimSpace(string(splitRunes[splitPos:]))
	}
	return lines
}

func (m *IntelligentMerger) canMerge(e1, e2 SubtitleEntry) (bool, string) {
	if e1.IsAudioEvent || e2.IsAudioEvent {
		return false, "audio event"
	}

	gap := e2.Start - e1.End
	if gap < m.MinSubtitleGap {
		return false, "gap too small"
	}
	if gap > 2.0 {
		return false, "gap too large"
	}

	mergedText := e1.Text + " " + e2.Text
	mergedDuration := e2.End - e1.Start

	maxAllowed := math.Min(m.MaxSubtitleDuration, 6.0)
	if mergedDuration > maxAllowed {
		return false, "duration too long"
	}

	mergedCPS := m.calculateCPS(mergedText, mergedDuration)
	dynamicLimit := m.getDynamicCPSLimit(mergedText)
	if mergedCPS > dynamicLimit {
		return false, "CPS too high"
	}

	mergedLines := m.calculateDisplayLines(mergedText)
	if mergedLines > 2 {
		return false, "too many lines"
	}

	if mergedLines == 1 && utf8.RuneCountInString(mergedText) > m.MaxCharsPerLine {
		return false, "single line too long"
	}

	return true, ""
}

func (m *IntelligentMerger) calculateMergeBenefit(e1, e2 SubtitleEntry) float64 {
	benefit := 0.0

	d1 := e1.End - e1.Start
	d2 := e2.End - e2.Start

	if d1 < m.MinSubtitleDuration {
		benefit += (m.MinSubtitleDuration - d1) * 20
	}
	if d2 < m.MinSubtitleDuration {
		benefit += (m.MinSubtitleDuration - d2) * 20
	}

	gap := e2.Start - e1.End
	if gap < 0.3 {
		benefit += (0.3 - gap) * 10
	} else if gap < 0.5 {
		benefit += (0.5 - gap) * 5
	}

	cc1 := e1.CharCount
	if cc1 == 0 {
		cc1 = utf8.RuneCountInString(e1.Text)
	}
	cc2 := e2.CharCount
	if cc2 == 0 {
		cc2 = utf8.RuneCountInString(e2.Text)
	}

	if cc1 < 3 {
		benefit += float64(3-cc1) * 5
	} else if cc1 < 8 {
		benefit += float64(8-cc1) * 2
	}

	if cc2 < 3 {
		benefit += float64(3-cc2) * 5
	} else if cc2 < 8 {
		benefit += float64(8-cc2) * 2
	}

	return benefit
}

func (m *IntelligentMerger) mergeTwoEntries(e1, e2 SubtitleEntry) SubtitleEntry {
	t1 := strings.TrimSpace(e1.Text)
	t2 := strings.TrimSpace(e2.Text)

	var mergedText string
	if t1 != "" && endsWithJoinPunctuation(t1) {
		mergedText = t1 + t2
	} else if m.IsCJK {
		mergedText = t1 + t2
	} else {
		mergedText = t1 + " " + t2
	}

	words := make([]Word, 0, len(e1.Words)+len(e2.Words))
	words = append(words, e1.Words...)
	words = append(words, e2.Words...)

	return SubtitleEntry{
		Text:         mergedText,
		Start:        e1.Start,
		End:          e2.End,
		Words:        words,
		IsAudioEvent: e1.IsAudioEvent || e2.IsAudioEvent,
		WordCount:    e1.WordCount + e2.WordCount,
		CharCount:    stripWhitespaceCount(mergedText),
	}
}

// MergeBasicEntries performs greedy forward merging of basic subtitle entries.
func (m *IntelligentMerger) MergeBasicEntries(entries []SubtitleEntry) []SubtitleEntry {
	if len(entries) == 0 {
		return nil
	}

	var merged []SubtitleEntry
	i := 0

	for i < len(entries) {
		current := entries[i]

		for i+1 < len(entries) {
			next := entries[i+1]

			canMerge, _ := m.canMerge(current, next)
			if !canMerge {
				break
			}

			benefit := m.calculateMergeBenefit(current, next)
			if benefit <= 5.0 {
				break
			}

			current = m.mergeTwoEntries(current, next)
			i++
		}

		merged = append(merged, current)
		i++
	}

	return merged
}

// OptimizeMergedEntries enforces min/max duration, CPS, and min gap constraints.
func (m *IntelligentMerger) OptimizeMergedEntries(entries []SubtitleEntry) []SubtitleEntry {
	if len(entries) == 0 {
		return nil
	}

	optimized := make([]SubtitleEntry, 0, len(entries))

	for i, entry := range entries {
		e := m.optimizeSingle(entry)

		// Ensure minimum gap with next entry.
		if i+1 < len(entries) {
			nextStart := entries[i+1].Start
			gap := nextStart - e.End
			if gap < m.MinSubtitleGap {
				e.End = nextStart - m.MinSubtitleGap
				minEnd := e.Start + m.MinSubtitleDuration
				if e.End < minEnd {
					e.End = minEnd
				}
			}
		}

		optimized = append(optimized, e)
	}

	return optimized
}

func (m *IntelligentMerger) optimizeSingle(entry SubtitleEntry) SubtitleEntry {
	e := entry
	duration := e.End - e.Start

	// Max duration.
	if duration > m.MaxSubtitleDuration {
		e.End = e.Start + m.MaxSubtitleDuration
		duration = m.MaxSubtitleDuration
	}

	// Min duration.
	if duration < m.MinSubtitleDuration {
		e.End = e.Start + m.MinSubtitleDuration
		duration = m.MinSubtitleDuration
	}

	// CPS check.
	cps := m.calculateCPS(e.Text, duration)
	dynamicLimit := m.getDynamicCPSLimit(e.Text)
	if cps > dynamicLimit {
		required := float64(stripWhitespaceCount(e.Text)) / dynamicLimit
		required = math.Min(required, m.MaxSubtitleDuration)
		e.End = e.Start + required
	}

	return e
}
