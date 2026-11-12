package main

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"strings"

	"github.com/Trendyzima/the-algorithm/testagram-feed/internal/ranking"
)

func topicRankHandler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !constantTimeBearer(r.Header.Get("Authorization"), token) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if r.Header.Get("X-Testagram-API-Version") != "1" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported or missing API version"})
			return
		}
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 128 || strings.ContainsAny(requestID, "\r\n") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Request-ID is required"})
			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "content type must be application/json"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var input ranking.TopicRequest
		if err := decoder.Decode(&input); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
			} else {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request must contain exactly one JSON value"})
			return
		}
		if len(input.Candidates) < 1 || len(input.Candidates) > 200 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "topic candidate count must be between 1 and 200"})
			return
		}
		if input.Limit < 1 || input.Limit > 50 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be between 1 and 50"})
			return
		}
		seen := make(map[string]struct{}, len(input.Candidates))
		for _, candidate := range input.Candidates {
			tag := strings.ToLower(strings.TrimSpace(candidate.Tag))
			if !ranking.ValidTopicTag(tag) || candidate.PostsCount < 1 || candidate.PostsCount > 1_000_000_000 ||
				candidate.LastUsedAt.IsZero() || math.IsNaN(candidate.InterestAffinity) ||
				math.IsInf(candidate.InterestAffinity, 0) || candidate.InterestAffinity < 0 || candidate.InterestAffinity > 1 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "topic candidates contain invalid tag, volume, timestamp or affinity"})
				return
			}
			if _, duplicate := seen[tag]; duplicate {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "duplicate topic tag"})
				return
			}
			seen[tag] = struct{}{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": ranking.RankTopics(input)})
	}
}
