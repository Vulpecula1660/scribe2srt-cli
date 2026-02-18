package config

// SubtitleSettings holds all subtitle generation parameters.
type SubtitleSettings struct {
	MinSubtitleDuration float64
	MaxSubtitleDuration float64
	MinSubtitleGap      float64
	CJKCPS              float64
	LatinCPS            float64
	CJKCharsPerLine     int
	LatinCharsPerLine   int
}

// Config holds the full application configuration.
type Config struct {
	SubtitleSettings

	SplitDurationMin    int
	MaxConcurrentChunks int
	MaxRetries          int
	APIRateLimitPerMin  int
}

// Default returns a Config with hardcoded defaults matching the Python version.
func Default() *Config {
	return &Config{
		SubtitleSettings: SubtitleSettings{
			MinSubtitleDuration: 0.83,
			MaxSubtitleDuration: 12.0,
			MinSubtitleGap:      0.083,
			CJKCPS:              11,
			LatinCPS:            15,
			CJKCharsPerLine:     25,
			LatinCharsPerLine:   42,
		},
		SplitDurationMin:    90,
		MaxConcurrentChunks: 3,
		MaxRetries:            3,
		APIRateLimitPerMin:    30,
	}
}
