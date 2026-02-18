package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"scribe2srt/internal/api"
	"scribe2srt/internal/config"
	"scribe2srt/internal/ffmpeg"
	"scribe2srt/internal/pipeline"
)

// applyTimeOffset adds an offset (in seconds) to all word timestamps, rounding to millisecond precision.
func applyTimeOffset(words []pipeline.Word, offsetSec float64) {
	for i := range words {
		words[i].Start = math.Round((words[i].Start+offsetSec)*1000) / 1000
		words[i].End = math.Round((words[i].End+offsetSec)*1000) / 1000
	}
}

// Options configures the worker.
type Options struct {
	InputPath        string
	OutputPath       string
	Language         string
	TagAudioEvents   bool
	NoAsync          bool
	MaxConcurrent    int
	MaxRetries       int
	RateLimitPerMin  int
	SplitDurationMin int
	SaveJSON         bool
	Settings         *config.SubtitleSettings
}

// Run is the top-level orchestrator for the transcription pipeline.
func Run(ctx context.Context, opts Options) error {
	inputPath := opts.InputPath

	// Determine output path.
	outputSRT := opts.OutputPath
	if outputSRT == "" {
		base := strings.TrimSuffix(inputPath, filepath.Ext(inputPath))
		outputSRT = base + ".srt"
	}

	slog.Info("processing file", "input", filepath.Base(inputPath))

	// Probe media.
	info := ffmpeg.LogMediaInfo(ctx, inputPath)
	duration := 0.0
	if info != nil {
		duration = info.Duration
	}

	splitDurationSec := opts.SplitDurationMin * 60
	workingPath := inputPath
	var tempAudioFile string

	// Extract audio from video if needed.
	ext := filepath.Ext(inputPath)
	if ffmpeg.IsVideoExtension(ext) && ffmpeg.Available() {
		base := strings.TrimSuffix(filepath.Base(inputPath), ext)
		tempAudioFile = filepath.Join(filepath.Dir(inputPath), "temp_audio_"+base+".m4a")
		slog.Info("extracting audio from video")
		if err := ffmpeg.ExtractAudio(ctx, inputPath, tempAudioFile); err != nil {
			return fmt.Errorf("extract audio: %w", err)
		}
		workingPath = tempAudioFile
		defer func() {
			os.Remove(tempAudioFile)
		}()
	}

	var combined *pipeline.TranscriptResponse
	var chunkFiles []string

	if duration > float64(splitDurationSec) && ffmpeg.Available() {
		// Split into chunks.
		slog.Info("file duration exceeds split threshold, splitting",
			"duration_min", int(duration/60), "threshold_min", opts.SplitDurationMin)

		chunks, err := ffmpeg.SplitAudio(ctx, workingPath, filepath.Dir(workingPath), splitDurationSec)
		if err != nil {
			return fmt.Errorf("split audio: %w", err)
		}
		chunkFiles = chunks
		defer cleanupChunks(chunkFiles)

		slog.Info("split into chunks", "count", len(chunks))

		if !opts.NoAsync && len(chunks) > 1 {
			combined, err = processConcurrent(ctx, chunks, splitDurationSec, opts)
		} else {
			combined, err = processSequential(ctx, chunks, splitDurationSec, opts)
		}
		if err != nil {
			return err
		}
	} else {
		// Single file processing.
		slog.Info("processing as single file")
		transcript, err := transcribeWithProgress(ctx, workingPath, opts)
		if err != nil {
			return fmt.Errorf("transcribe: %w", err)
		}
		combined = transcript
	}

	if combined == nil || (len(combined.Words) == 0 && combined.Text == "") {
		return fmt.Errorf("empty transcript received")
	}

	// Save combined JSON if requested.
	if opts.SaveJSON {
		jsonPath := strings.TrimSuffix(outputSRT, filepath.Ext(outputSRT)) + ".json"
		if err := saveJSON(jsonPath, combined); err != nil {
			slog.Warn("failed to save JSON", "err", err)
		} else {
			slog.Info("transcript JSON saved", "path", jsonPath)
		}
	}

	// Generate SRT.
	slog.Info("generating SRT subtitles")
	srtContent := pipeline.Process(combined, opts.Settings)
	if srtContent == "" {
		return fmt.Errorf("SRT generation produced empty output")
	}

	if err := os.WriteFile(outputSRT, []byte(srtContent), 0644); err != nil {
		return fmt.Errorf("write SRT file: %w", err)
	}

	slog.Info("SRT file saved", "path", outputSRT)
	return nil
}

func transcribeWithProgress(ctx context.Context, path string, opts Options) (*pipeline.TranscriptResponse, error) {
	progress := func(read, total int64) {
		pct := 0.0
		if total > 0 {
			pct = math.Min(float64(read)/float64(total)*100, 100)
		}
		slog.Debug("upload progress", "percent", fmt.Sprintf("%.1f%%", pct))
	}

	return api.Transcribe(ctx, path, opts.Language, opts.TagAudioEvents, progress)
}

func saveJSON(path string, transcript *pipeline.TranscriptResponse) error {
	data, err := json.MarshalIndent(transcript, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func cleanupChunks(chunks []string) {
	for _, chunk := range chunks {
		if err := os.Remove(chunk); err != nil && !os.IsNotExist(err) {
			slog.Debug("cleanup chunk", "file", filepath.Base(chunk), "err", err)
		}
		// Also remove companion JSON files.
		jsonPath := strings.TrimSuffix(chunk, filepath.Ext(chunk)) + ".json"
		os.Remove(jsonPath)
	}
	slog.Debug("temp chunk cleanup complete")
}
