package pipeline

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"scribe2srt/internal/config"
)

// Process runs the full two-stage subtitle pipeline on a transcript and
// returns the SRT content string.
func Process(transcript *TranscriptResponse, settings *config.SubtitleSettings) string {
	langCode := transcript.LanguageCode
	if len(langCode) > 3 {
		langCode = langCode[:3]
	}
	isCJK := config.IsCJK(langCode)

	maxCPL := settings.LatinCharsPerLine
	if isCJK {
		maxCPL = settings.CJKCharsPerLine
	}

	// Preprocess words.
	result := preprocessWords(transcript.Words)

	if len(result.Words) == 0 && len(result.AudioEvents) == 0 {
		return ""
	}

	// Stage 1: sentence splitting.
	var basicEntries []SubtitleEntry
	if len(result.Words) > 0 {
		splitter := NewSentenceSplitter(langCode)
		groups := splitter.SplitIntoSentenceGroups(result.Words)
		basicEntries = splitter.CreateBasicEntries(groups)
	}

	// Audio event entries.
	audioEntries := createAudioEventEntries(result.AudioEvents)

	// Stage 2: intelligent merging.
	var mergedEntries []SubtitleEntry
	if len(basicEntries) > 0 {
		// Build subtitle settings for the merger, matching Python's create_srt logic.
		mergerSettings := &config.SubtitleSettings{
			MinSubtitleDuration: settings.MinSubtitleDuration,
			MaxSubtitleDuration: settings.MaxSubtitleDuration,
			MinSubtitleGap:      settings.MinSubtitleGap,
		}
		if isCJK {
			mergerSettings.CJKCPS = settings.CJKCPS
			mergerSettings.LatinCPS = config.CPSForLang("en")
			mergerSettings.CJKCharsPerLine = settings.CJKCharsPerLine
			mergerSettings.LatinCharsPerLine = config.CPLForLang("en")
		} else {
			mergerSettings.CJKCPS = config.CPSForLang("zh")
			mergerSettings.LatinCPS = settings.LatinCPS
			mergerSettings.CJKCharsPerLine = config.CPLForLang("zh")
			mergerSettings.LatinCharsPerLine = settings.LatinCharsPerLine
		}

		merger := NewIntelligentMerger(langCode, mergerSettings)
		mergedEntries = merger.MergeBasicEntries(basicEntries)
		mergedEntries = merger.OptimizeMergedEntries(mergedEntries)
	}

	// Combine and sort.
	all := append(mergedEntries, audioEntries...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Start < all[j].Start
	})

	// Generate SRT.
	return generateSRT(all, maxCPL)
}

func createAudioEventEntries(events []Word) []SubtitleEntry {
	entries := make([]SubtitleEntry, 0, len(events))
	for _, ev := range events {
		charCount := utf8.RuneCountInString(strings.ReplaceAll(ev.Text, " ", ""))
		entries = append(entries, SubtitleEntry{
			Text:         ev.Text,
			Start:        ev.Start,
			End:          ev.End,
			Words:        []Word{ev},
			IsAudioEvent: true,
			WordCount:    0,
			CharCount:    charCount,
		})
	}
	return entries
}

func generateSRT(entries []SubtitleEntry, maxCPL int) string {
	if len(entries) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, entry := range entries {
		startStr := formatSRTTime(entry.Start)
		endStr := formatSRTTime(entry.End)
		text := optimizeTextDisplay(entry.Text, maxCPL)

		fmt.Fprintf(&sb, "%d\n%s --> %s\n%s\n", i+1, startStr, endStr, text)
		if i < len(entries)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
