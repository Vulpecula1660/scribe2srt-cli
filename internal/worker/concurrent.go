package worker

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"scribe2srt/internal/api"
	"scribe2srt/internal/pipeline"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

type chunkResult struct {
	Index      int
	Transcript *pipeline.TranscriptResponse
}

// processConcurrent processes chunks concurrently with bounded parallelism and rate limiting.
func processConcurrent(ctx context.Context, chunks []string, splitDurationSec int, opts Options) (*pipeline.TranscriptResponse, error) {
	slog.Info("starting concurrent processing",
		"chunks", len(chunks),
		"max_concurrent", opts.MaxConcurrent,
		"rate_limit_rpm", opts.RateLimitPerMin)

	// Rate limiter: tokens per second = RPM / 60.
	limiter := rate.NewLimiter(rate.Limit(float64(opts.RateLimitPerMin)/60.0), 1)

	var (
		mu      sync.Mutex
		results []chunkResult
	)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(opts.MaxConcurrent)

	for i, chunk := range chunks {
		g.Go(func() error {
			// Rate limit.
			if err := limiter.Wait(gctx); err != nil {
				return fmt.Errorf("rate limiter: %w", err)
			}

			slog.Info("starting chunk upload", "chunk", fmt.Sprintf("%d/%d", i+1, len(chunks)))

			var transcript *pipeline.TranscriptResponse
			var lastErr error

			// Retry loop with exponential backoff.
			for attempt := 0; attempt < opts.MaxRetries; attempt++ {
				select {
				case <-gctx.Done():
					return gctx.Err()
				default:
				}

				progress := func(read, total int64) {
					pct := 0.0
					if total > 0 {
						pct = math.Min(float64(read)/float64(total)*100, 100)
					}
					slog.Debug("chunk upload progress",
						"chunk", i+1,
						"percent", fmt.Sprintf("%.1f%%", pct))
				}

				t, err := api.Transcribe(gctx, chunk, opts.Language, opts.TagAudioEvents, progress)
				if err == nil {
					transcript = t
					break
				}

				lastErr = err
				if attempt < opts.MaxRetries-1 {
					backoff := 1 << uint(attempt) // 1s, 2s, 4s...
					slog.Warn("chunk failed, retrying",
						"chunk", i+1,
						"attempt", attempt+1,
						"backoff_sec", backoff,
						"err", err)

					timer := time.NewTimer(time.Duration(backoff) * time.Second)
					select {
					case <-gctx.Done():
						timer.Stop()
						return gctx.Err()
					case <-timer.C:
					}
				}
			}

			if transcript == nil {
				return fmt.Errorf("chunk %d/%d failed after %d retries: %w",
					i+1, len(chunks), opts.MaxRetries, lastErr)
			}

			// Apply time offset.
			if i > 0 {
				applyTimeOffset(transcript.Words, float64(i*splitDurationSec))
			}

			mu.Lock()
			results = append(results, chunkResult{Index: i, Transcript: transcript})
			mu.Unlock()

			slog.Info("chunk completed", "chunk", fmt.Sprintf("%d/%d", i+1, len(chunks)))
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		// Check if some chunks succeeded — try fallback to sequential for remaining.
		mu.Lock()
		completedCount := len(results)
		mu.Unlock()

		if completedCount > 0 {
			slog.Warn("concurrent processing partially failed, falling back to sequential",
				"completed", completedCount, "total", len(chunks), "err", err)
			return fallbackToSequential(ctx, chunks, splitDurationSec, opts, results)
		}
		return nil, err
	}

	return mergeResults(results), nil
}

func mergeResults(results []chunkResult) *pipeline.TranscriptResponse {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Index < results[j].Index
	})

	combined := &pipeline.TranscriptResponse{
		LanguageCode: results[0].Transcript.LanguageCode,
	}

	for _, r := range results {
		combined.Words = append(combined.Words, r.Transcript.Words...)
		if combined.Text != "" {
			combined.Text += " "
		}
		combined.Text += r.Transcript.Text
	}

	return combined
}

func fallbackToSequential(ctx context.Context, chunks []string, splitDurationSec int, opts Options, completed []chunkResult) (*pipeline.TranscriptResponse, error) {
	slog.Info("falling back to sequential processing for remaining chunks")

	// Track which chunks are done.
	done := make(map[int]bool)
	for _, r := range completed {
		done[r.Index] = true
	}

	// Process remaining chunks sequentially.
	for i, chunk := range chunks {
		if done[i] {
			continue
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		slog.Info("sequential fallback processing chunk", "chunk", fmt.Sprintf("%d/%d", i+1, len(chunks)))

		transcript, err := transcribeWithProgress(ctx, chunk, opts)
		if err != nil {
			return nil, fmt.Errorf("sequential fallback chunk %d/%d: %w", i+1, len(chunks), err)
		}

		if i > 0 {
			applyTimeOffset(transcript.Words, float64(i*splitDurationSec))
		}

		completed = append(completed, chunkResult{Index: i, Transcript: transcript})
	}

	return mergeResults(completed), nil
}
