package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scribe2srt/internal/pipeline"
)

const (
	sttAPIURL     = "https://api.elevenlabs.io/v1/speech-to-text"
	sttModelID    = "scribe_v2"
	uploadTimeout = 30 * time.Minute
)

// ProgressFunc is called with (bytesRead, totalBytes) during upload.
type ProgressFunc func(bytesRead, totalBytes int64)

// progressReader wraps an io.Reader and reports progress.
type progressReader struct {
	reader   io.Reader
	total    int64
	read     int64
	callback ProgressFunc
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.callback != nil {
		pr.callback(pr.read, pr.total)
	}
	return n, err
}

// mimeFromExt returns the MIME type for common audio/video extensions.
func mimeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp3":
		return "audio/mp3"
	case ".m4a":
		return "audio/m4a"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".aac":
		return "audio/aac"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/mov"
	default:
		return "application/octet-stream"
	}
}

// Transcribe uploads an audio/video file to ElevenLabs STT and returns the transcript.
func Transcribe(ctx context.Context, filePath, languageCode string, tagAudioEvents bool, progress ProgressFunc) (*pipeline.TranscriptResponse, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	fileSize := stat.Size()

	// Build multipart form body using a pipe.
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	// Write form fields and file in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		defer mw.Close()

		if err := mw.WriteField("model_id", sttModelID); err != nil {
			errCh <- err
			return
		}
		if err := mw.WriteField("diarize", "true"); err != nil {
			errCh <- err
			return
		}
		tagStr := "false"
		if tagAudioEvents {
			tagStr = "true"
		}
		if err := mw.WriteField("tag_audio_events", tagStr); err != nil {
			errCh <- err
			return
		}
		if languageCode != "" && strings.ToLower(languageCode) != "auto" {
			if err := mw.WriteField("language_code", languageCode); err != nil {
				errCh <- err
				return
			}
		}

		mimeType := mimeFromExt(filepath.Ext(filePath))
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(filePath)))
		h.Set("Content-Type", mimeType)
		part, err := mw.CreatePart(h)
		if err != nil {
			errCh <- err
			return
		}

		if _, err := io.Copy(part, f); err != nil {
			errCh <- err
			return
		}

		errCh <- nil
	}()

	// Wrap the reader for progress reporting.
	// Estimate total size: file size + ~1KB form overhead.
	estimatedTotal := fileSize + 1024
	body := &progressReader{
		reader:   pr,
		total:    estimatedTotal,
		callback: progress,
	}

	// Build request.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sttAPIURL+"?allow_unauthenticated=1", body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	headers := RandomHeaders()
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	client := &http.Client{Timeout: uploadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for write error.
	if writeErr := <-errCh; writeErr != nil {
		return nil, fmt.Errorf("multipart write error: %w", writeErr)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var transcript pipeline.TranscriptResponse
	if err := json.NewDecoder(resp.Body).Decode(&transcript); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &transcript, nil
}
