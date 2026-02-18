package api

import (
	"math/rand/v2"
	"net/http"
)

// Browser User-Agent strings for request spoofing.
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:127.0) Gecko/20100101 Firefox/127.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
}

// Accept-Language header values.
var acceptLanguages = []string{
	"zh-CN,zh;q=0.9,en;q=0.8",
	"en-US,en;q=0.9,es;q=0.8",
	"en-GB,en;q=0.9",
	"ja-JP,ja;q=0.9,en;q=0.8",
	"ko-KR,ko;q=0.9,en;q=0.8",
	"de-DE,de;q=0.9,en;q=0.8",
	"fr-FR,fr;q=0.9,en;q=0.8",
	"en-US,en;q=0.5",
}

// Common request headers.
var baseHeaders = map[string]string{
	"accept": "*/*",
	// Note: Do NOT set Accept-Encoding manually. Go's http.Transport handles
	// gzip automatically and transparently decompresses the response body,
	// but only when Accept-Encoding is not set by the caller.
	"origin":         "https://elevenlabs.io",
	"referer":         "https://elevenlabs.io/",
	"sec-fetch-dest":  "empty",
	"sec-fetch-mode":  "cors",
	"sec-fetch-site":  "same-site",
}

// RandomHeaders returns an http.Header with randomized User-Agent and Accept-Language.
func RandomHeaders() http.Header {
	h := make(http.Header)
	for k, v := range baseHeaders {
		h.Set(k, v)
	}
	h.Set("User-Agent", userAgents[rand.IntN(len(userAgents))])
	h.Set("Accept-Language", acceptLanguages[rand.IntN(len(acceptLanguages))])
	return h
}
