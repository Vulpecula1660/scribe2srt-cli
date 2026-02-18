package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// MediaInfo holds duration and codec information from ffprobe.
type MediaInfo struct {
	Duration float64
	Codec    string
}

// Available returns true if ffmpeg is on the PATH.
func Available() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// probeOutput mirrors ffprobe JSON structure.
type probeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecName string `json:"codec_name"`
	} `json:"streams"`
}

// ProbeMedia uses ffprobe to get media duration and audio codec.
func ProbeMedia(ctx context.Context, path string) (*MediaInfo, error) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, fmt.Errorf("ffprobe not found: %w", err)
	}

	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_name:format=duration",
		"-of", "json",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probe probeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, fmt.Errorf("ffprobe JSON parse error: %w", err)
	}

	dur, _ := strconv.ParseFloat(probe.Format.Duration, 64)

	codec := "N/A"
	if len(probe.Streams) > 0 && probe.Streams[0].CodecName != "" {
		codec = probe.Streams[0].CodecName
	}

	return &MediaInfo{Duration: dur, Codec: codec}, nil
}

// ExtractAudio extracts the audio stream from a video file using ffmpeg -vn -c:a copy.
func ExtractAudio(ctx context.Context, videoPath, outputPath string) error {
	slog.Info("extracting audio", "input", filepath.Base(videoPath), "output", filepath.Base(outputPath))

	cmd := exec.CommandContext(ctx,
		"ffmpeg", "-i", videoPath,
		"-vn", "-c:a", "copy", "-y",
		outputPath,
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg extract audio failed: %w\n%s", err, string(out))
	}
	return nil
}

// SplitAudio splits an audio file into segments of segmentSec seconds using ffmpeg.
// Returns the sorted list of chunk file paths.
func SplitAudio(ctx context.Context, audioPath string, outputDir string, segmentSec int) ([]string, error) {
	baseName := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	outputTemplate := filepath.Join(outputDir, baseName+"_chunk_%03d.mp3")

	slog.Info("splitting audio", "file", filepath.Base(audioPath), "segment_sec", segmentSec)

	cmd := exec.CommandContext(ctx,
		"ffmpeg", "-i", audioPath,
		"-f", "segment",
		"-segment_time", strconv.Itoa(segmentSec),
		"-c:a", "libmp3lame",
		"-b:a", "192k",
		"-y",
		outputTemplate,
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg split failed: %w\n%s", err, string(out))
	}

	// Collect generated chunk files.
	pattern := filepath.Join(outputDir, baseName+"_chunk_*.mp3")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob chunk files: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no chunk files")
	}

	sort.Strings(matches)
	return matches, nil
}

// IsVideoExtension returns true for common video file extensions.
func IsVideoExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".mp4", ".mkv", ".mov", ".avi", ".flv", ".webm":
		return true
	}
	return false
}

// LogMediaInfo logs file size and media information.
func LogMediaInfo(ctx context.Context, path string) *MediaInfo {
	stat, err := os.Stat(path)
	if err != nil {
		slog.Warn("cannot stat file", "path", path, "err", err)
		return nil
	}

	sizeMB := float64(stat.Size()) / (1024 * 1024)
	msg := fmt.Sprintf("file size: %.2f MB", sizeMB)

	info, err := ProbeMedia(ctx, path)
	if err == nil && info != nil {
		minutes := int(info.Duration) / 60
		seconds := int(info.Duration) % 60
		msg += fmt.Sprintf(" | duration: %02d:%02d | codec: %s", minutes, seconds, info.Codec)
	}

	slog.Info(msg)
	return info
}
