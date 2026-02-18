package pipeline

// Word represents a single word/token from the ElevenLabs transcript.
type Word struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Type  string  `json:"type"` // "word", "spacing", "audio_event"
}

// SubtitleEntry represents one subtitle block.
type SubtitleEntry struct {
	Text         string
	Start        float64
	End          float64
	Words        []Word
	IsAudioEvent bool
	WordCount    int
	CharCount    int
}

// TranscriptResponse is the top-level JSON structure from ElevenLabs.
type TranscriptResponse struct {
	LanguageCode string `json:"language_code"`
	Text         string `json:"text"`
	Words        []Word `json:"words"`
}
