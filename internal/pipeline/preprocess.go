package pipeline

import (
	"strings"
	"unicode/utf8"
)

// cjkMergePunctuation is the set of standalone CJK punctuation that should be
// merged into the preceding word during preprocessing.
var cjkMergePunctuation = map[rune]struct{}{
	'\u3002': {}, // 。
	'\uff1f': {}, // ？
	'\uff01': {}, // ！
	'\u300d': {}, // 」
	'\u300c': {}, // 「
	'\u3001': {}, // 、
	'\u30fb': {}, // ・
	'\uff0c': {}, // ，
}

// preprocessResult holds the output of preprocessing.
type preprocessResult struct {
	Words       []Word
	AudioEvents []Word
}

// preprocessWords separates audio events, drops spacing tokens (appending a
// space to the previous word), and merges standalone CJK punctuation into the
// preceding word. This mirrors SrtProcessor._preprocess_words.
func preprocessWords(raw []Word) preprocessResult {
	var words []Word
	var audioEvents []Word

	for _, w := range raw {
		// Audio events go into a separate slice.
		if w.Type == "audio_event" {
			audioEvents = append(audioEvents, w)
			continue
		}

		// Spacing tokens: append space to previous word and skip.
		if w.Type == "spacing" {
			if len(words) > 0 &&
				strings.TrimSpace(w.Text) == "" &&
				words[len(words)-1].Type == "word" &&
				!strings.HasSuffix(words[len(words)-1].Text, " ") {
				words[len(words)-1].Text += " "
			}
			continue
		}

		// Merge standalone CJK punctuation into previous word.
		runes := []rune(w.Text)
		if len(runes) == 1 {
			if _, ok := cjkMergePunctuation[runes[0]]; ok && len(words) > 0 {
				prev := &words[len(words)-1]
				if prev.Type == "word" && prev.Text != "" {
					lastRune, _ := utf8.DecodeLastRuneInString(prev.Text)
					if _, isCJKPunct := cjkMergePunctuation[lastRune]; !isCJKPunct {
						prev.Text += w.Text
						prev.End = w.End
						continue
					}
				}
			}
		}

		words = append(words, w)
	}

	return preprocessResult{Words: words, AudioEvents: audioEvents}
}
