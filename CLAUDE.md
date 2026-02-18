# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
make build          # Build binary (stripped)
make run ARGS="transcribe input.mp4 -l ja"  # Run without compiling to disk
make test           # Run all tests (go test ./...)
make clean          # Remove compiled binary
```

Single test: `go test ./internal/pipeline/ -run TestName`

## Architecture

Go CLI tool that converts audio/video files to SRT subtitles using the ElevenLabs Scribe v2 Speech-to-Text API (unauthenticated endpoint with browser-like header spoofing).

### Data Flow

```
Input file
  → [ffmpeg] Extract audio from video if needed, split into chunks if > 90min
  → [worker] Process chunks (concurrent with errgroup or sequential)
      → [api] Upload to ElevenLabs STT, with retry + rate limiting
  → [pipeline] Two-stage subtitle processing:
      Stage 0: PreprocessWords — separate audio events, merge spacing/CJK punctuation
      Stage 1: SentenceSplitter — split into sentence groups by punctuation priority
      Stage 2: IntelligentMerger — greedy forward merging + post-merge optimization
  → Output .srt file
```

### Package Layout

- **`cmd/`** — Cobra CLI command (transcribe). Root sets up logging.
- **`internal/api/`** — ElevenLabs HTTP client. Multipart streaming upload with progress. Randomized browser headers.
- **`internal/config/`** — `SubtitleSettings` and `Config` structs with defaults. CJK vs Latin language detection and per-script CPS/CPL constants.
- **`internal/ffmpeg/`** — Wraps `ffmpeg`/`ffprobe` CLI for audio extraction, splitting, and media probing.
- **`internal/pipeline/`** — Core subtitle processing. Three-tier punctuation priority system (high/medium/low) drives sentence splitting. Merger uses benefit scoring to combine short/close entries while enforcing duration, CPS, CPL, and gap constraints.
- **`internal/worker/`** — Orchestrates the full flow: media handling → API calls → pipeline → file output. Concurrent mode uses `errgroup` with bounded parallelism and `rate.Limiter`.

### Key Design Notes

- Ported from Python — comments reference matching Python predecessor behavior (e.g., `_preprocess_words`, `_merge_two_entries`).
- CJK-aware throughout: punctuation merging, line splitting, and separate CPS/CPL defaults for CJK (11 CPS, 25 CPL) vs Latin (15 CPS, 42 CPL) scripts.
- Entry point is `main.go` which calls `cmd.Execute()`.
