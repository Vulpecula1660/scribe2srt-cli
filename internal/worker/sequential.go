package worker

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"scribe2srt/internal/pipeline"
)

// processSequential processes chunks one at a time, applying time offsets.
func processSequential(ctx context.Context, chunks []string, splitDurationSec int, opts Options) (*pipeline.TranscriptResponse, error) {
	var combined *pipeline.TranscriptResponse

	for i, chunk := range chunks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		slog.Info("processing chunk",
			"chunk", fmt.Sprintf("%d/%d", i+1, len(chunks)),
			"file", filepath.Base(chunk))

		transcript, err := transcribeWithProgress(ctx, chunk, opts)
		if err != nil {
			return nil, fmt.Errorf("chunk %d/%d failed: %w", i+1, len(chunks), err)
		}

		// Apply time offset to words (skip first chunk — offset is 0).
		if i > 0 {
			applyTimeOffset(transcript.Words, float64(i*splitDurationSec))
		}

		if combined == nil {
			combined = transcript
		} else {
			combined.Words = append(combined.Words, transcript.Words...)
			if combined.Text != "" {
				combined.Text += " "
			}
			combined.Text += transcript.Text
		}

		slog.Info("chunk completed", "chunk", fmt.Sprintf("%d/%d", i+1, len(chunks)))
	}

	return combined, nil
}
