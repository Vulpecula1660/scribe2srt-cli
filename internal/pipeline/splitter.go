package pipeline

import (
	"strings"
	"unicode/utf8"

	"scribe2srt/internal/config"
)

// SentenceSplitter implements Stage 1 of the subtitle pipeline.
type SentenceSplitter struct {
	Language string
	IsCJK   bool
}

// NewSentenceSplitter creates a new splitter for the given language code.
func NewSentenceSplitter(langCode string) *SentenceSplitter {
	lang := langCode
	if len(lang) > 3 {
		lang = lang[:3]
	}
	return &SentenceSplitter{
		Language: lang,
		IsCJK:   config.IsCJK(lang),
	}
}

// shouldSplitAtWord decides whether to split after the current word.
func (s *SentenceSplitter) shouldSplitAtWord(word Word, accumulated []Word) bool {
	text := strings.TrimSpace(word.Text)
	hasPunct, _, priority := wordEndsWithPunctuation(text)
	if !hasPunct {
		return false
	}

	// High priority → always split.
	if priority == priorityHigh {
		return true
	}

	// Medium priority → split if >= 3 accumulated words.
	if priority == priorityMedium {
		if len(accumulated) >= 3 {
			return true
		}
	}

	// Low priority → split if >= 5 words AND >= 15 chars.
	if priority == priorityLow {
		if len(accumulated) >= 5 {
			totalChars := 0
			for _, w := range accumulated {
				totalChars += utf8.RuneCountInString(w.Text)
			}
			if totalChars >= 15 {
				return true
			}
		}
	}

	return false
}

// SplitIntoSentenceGroups splits a preprocessed word list into sentence groups.
func (s *SentenceSplitter) SplitIntoSentenceGroups(words []Word) [][]Word {
	if len(words) == 0 {
		return nil
	}

	var groups [][]Word
	var current []Word

	for i, word := range words {
		current = append(current, word)

		// accumulated = current minus the last element (the current word).
		accumulated := current[:len(current)-1]
		shouldSplit := s.shouldSplitAtWord(word, accumulated)
		isLast := i == len(words)-1

		if shouldSplit || isLast {
			if len(current) > 0 {
				groups = append(groups, current)
				current = nil
			}
		}
	}

	if len(current) > 0 {
		groups = append(groups, current)
	}

	return groups
}

// CreateBasicEntries converts sentence groups into SubtitleEntry values.
func (s *SentenceSplitter) CreateBasicEntries(groups [][]Word) []SubtitleEntry {
	var entries []SubtitleEntry

	for _, group := range groups {
		if len(group) == 0 {
			continue
		}

		// Collect actual words (type == "word") for timing.
		var actualWords []Word
		for _, w := range group {
			if w.Type == "word" {
				actualWords = append(actualWords, w)
			}
		}
		if len(actualWords) == 0 {
			continue
		}

		// Build text from all words in the group.
		var b strings.Builder
		for _, w := range group {
			b.WriteString(w.Text)
		}
		text := strings.TrimSpace(b.String())
		if text == "" {
			continue
		}

		charCount := utf8.RuneCountInString(strings.ReplaceAll(text, " ", ""))

		entries = append(entries, SubtitleEntry{
			Text:         text,
			Start:        actualWords[0].Start,
			End:          actualWords[len(actualWords)-1].End,
			Words:        group,
			IsAudioEvent: false,
			WordCount:    len(actualWords),
			CharCount:    charCount,
		})
	}

	return entries
}
