package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"scribe2srt/internal/config"
	"scribe2srt/internal/worker"

	"github.com/spf13/cobra"
)

var transcribeCmd = &cobra.Command{
	Use:   "transcribe <input-file>",
	Short: "Transcribe audio/video to SRT subtitles",
	Long: `Transcribe an audio or video file into an SRT subtitle file using the
ElevenLabs Speech-to-Text API with a two-stage processing pipeline.`,
	Args: cobra.ExactArgs(1),
	RunE: runTranscribe,
}

var (
	language        string
	output          string
	tagAudioEvents  bool
	noAsync         bool
	maxConcurrent   int
	maxRetries      int
	rateLimit       int
	splitDuration   int
	saveJSON        bool

	// Subtitle tuning flags.
	minDuration   float64
	maxDuration   float64
	minGap        float64
	cjkCPS        float64
	latinCPS      float64
	cjkCPL        int
	latinCPL      int
)

func init() {
	defaults := config.Default()

	transcribeCmd.Flags().StringVarP(&language, "language", "l", "auto", "language: ko, ja, zh, en, auto")
	transcribeCmd.Flags().StringVarP(&output, "output", "o", "", "output SRT path (default: <input>.srt)")
	transcribeCmd.Flags().BoolVar(&tagAudioEvents, "tag-audio-events", true, "tag audio events")
	transcribeCmd.Flags().BoolVar(&noAsync, "no-async", false, "disable concurrent chunk processing")
	transcribeCmd.Flags().IntVarP(&maxConcurrent, "max-concurrent", "j", defaults.MaxConcurrentChunks, "max concurrent API uploads")
	transcribeCmd.Flags().IntVar(&maxRetries, "max-retries", defaults.MaxRetries, "max retries per chunk")
	transcribeCmd.Flags().IntVar(&rateLimit, "rate-limit", defaults.APIRateLimitPerMin, "API requests per minute")
	transcribeCmd.Flags().IntVar(&splitDuration, "split-duration", defaults.SplitDurationMin, "audio split threshold in minutes")
	transcribeCmd.Flags().BoolVar(&saveJSON, "save-json", false, "save combined transcript JSON alongside SRT")

	// Subtitle tuning flags.
	transcribeCmd.Flags().Float64Var(&minDuration, "min-duration", defaults.MinSubtitleDuration, "minimum subtitle duration in seconds")
	transcribeCmd.Flags().Float64Var(&maxDuration, "max-duration", defaults.MaxSubtitleDuration, "maximum subtitle duration in seconds")
	transcribeCmd.Flags().Float64Var(&minGap, "min-gap", defaults.MinSubtitleGap, "minimum gap between subtitles in seconds")
	transcribeCmd.Flags().Float64Var(&cjkCPS, "cjk-cps", defaults.CJKCPS, "CJK characters per second limit")
	transcribeCmd.Flags().Float64Var(&latinCPS, "latin-cps", defaults.LatinCPS, "Latin characters per second limit")
	transcribeCmd.Flags().IntVar(&cjkCPL, "cjk-cpl", defaults.CJKCharsPerLine, "CJK characters per line limit")
	transcribeCmd.Flags().IntVar(&latinCPL, "latin-cpl", defaults.LatinCharsPerLine, "Latin characters per line limit")

	rootCmd.AddCommand(transcribeCmd)
}

func runTranscribe(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	// Resolve to absolute path.
	absPath, err := filepath.Abs(inputPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", inputPath)
	}

	// Validate file extension.
	ext := strings.ToLower(filepath.Ext(absPath))
	validExts := map[string]bool{
		".mp3": true, ".m4a": true, ".wav": true, ".flac": true,
		".ogg": true, ".aac": true, ".mp4": true, ".mov": true,
		".mkv": true, ".avi": true, ".flv": true, ".webm": true,
	}
	if !validExts[ext] {
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	settings := &config.SubtitleSettings{
		MinSubtitleDuration: minDuration,
		MaxSubtitleDuration: maxDuration,
		MinSubtitleGap:      minGap,
		CJKCPS:              cjkCPS,
		LatinCPS:            latinCPS,
		CJKCharsPerLine:     cjkCPL,
		LatinCharsPerLine:   latinCPL,
	}

	// Setup signal handling for graceful cancellation.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	opts := worker.Options{
		InputPath:        absPath,
		OutputPath:       output,
		Language:         language,
		TagAudioEvents:   tagAudioEvents,
		NoAsync:          noAsync,
		MaxConcurrent:    maxConcurrent,
		MaxRetries:       maxRetries,
		RateLimitPerMin:  rateLimit,
		SplitDurationMin: splitDuration,
		SaveJSON:         saveJSON,
		Settings:         settings,
	}

	if err := worker.Run(ctx, opts); err != nil {
		return err
	}

	if !quiet {
		slog.Info("done")
	}
	return nil
}
