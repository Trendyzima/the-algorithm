package ranking

import (
	"math"
	"testing"
	"time"
)

func TestRankFiltersUnsafeAndSeenCandidates(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	in := Request{
		Now: now, SeenIDs: []string{"seen"},
		BlockedAuthorIDs: []string{"blocked"}, MutedAuthorIDs: []string{"muted"},
		Candidates: []Candidate{
			{ID: "seen", AuthorID: "a", Visible: true, CreatedAt: now},
			{ID: "blocked-post", AuthorID: "blocked", Visible: true, CreatedAt: now},
			{ID: "muted-post", AuthorID: "muted", Visible: true, CreatedAt: now},
			{ID: "hidden", AuthorID: "a", Visible: false, CreatedAt: now},
			{ID: "risky", AuthorID: "a", Visible: true, CreatedAt: now, Features: Features{ReportRisk: .9}},
			{ID: "safe", AuthorID: "b", Visible: true, CreatedAt: now, Features: Features{Quality: .8}},
		},
	}
	got := Rank(in)
	if len(got) != 1 || got[0].ID != "safe" {
		t.Fatalf("unexpected ranked results: %#v", got)
	}
}

func TestRankCapsAuthorDominanceAndIsDeterministic(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	in := Request{Now: now, Limit: 10, Candidates: []Candidate{
		{ID: "a1", AuthorID: "author-a", Visible: true, CreatedAt: now, Features: Features{Quality: 1}},
		{ID: "a2", AuthorID: "author-a", Visible: true, CreatedAt: now.Add(-time.Minute), Features: Features{Quality: 1}},
		{ID: "a3", AuthorID: "author-a", Visible: true, CreatedAt: now.Add(-2 * time.Minute), Features: Features{Quality: 1}},
		{ID: "b1", AuthorID: "author-b", Visible: true, CreatedAt: now, Features: Features{Quality: 1}},
	}}
	first, second := Rank(in), Rank(in)
	if len(first) != 4 {
		t.Fatalf("expected the requested page to be filled after diversity pass, got %d", len(first))
	}
	if first[0].ID != second[0].ID || first[1].ID != second[1].ID {
		t.Fatal("ranking must be deterministic")
	}
	authorACount := 0
	for _, item := range first {
		if item.AuthorID == "author-a" {
			authorACount++
		}
	}
	if authorACount != 3 {
		t.Fatalf("expected the diversity pass to prioritize two author-a posts and then fill remaining slots, got %d", authorACount)
	}
	firstPassAuthorACount := 0
	for _, item := range first[:3] {
		if item.AuthorID == "author-a" {
			firstPassAuthorACount++
		}
	}
	if firstPassAuthorACount != 2 {
		t.Fatalf("first diversity pass included %d author-a posts, want 2", firstPassAuthorACount)
	}
}

func TestRankClampsInvalidFeatures(t *testing.T) {
	now := time.Now().UTC()
	got := Rank(Request{Now: now, Candidates: []Candidate{
		{ID: "x", AuthorID: "u", Visible: true, CreatedAt: now, Features: Features{Quality: 1000, InterestAffinity: -4}},
	}})
	if len(got) != 1 || got[0].Score < 0 || got[0].Score > 1 {
		t.Fatalf("score out of range: %#v", got)
	}
}

func TestRankRejectsMalformedAndDuplicateCandidateIDs(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	got := Rank(Request{Now: now, Limit: 10, Candidates: []Candidate{
		{ID: "post-1", AuthorID: "author-1", Visible: true, CreatedAt: now},
		{ID: "post-1", AuthorID: "author-2", Visible: true, CreatedAt: now},
		{ID: " post-2", AuthorID: "author-2", Visible: true, CreatedAt: now},
		{ID: "post-3 ", AuthorID: "author-3", Visible: true, CreatedAt: now},
		{ID: "post-4", AuthorID: " author-4", Visible: true, CreatedAt: now},
		{ID: "post-5", AuthorID: "author-5 ", Visible: true, CreatedAt: now},
		{ID: string(make([]byte, 129)), AuthorID: "author-6", Visible: true, CreatedAt: now},
	}})
	if len(got) != 1 || got[0].ID != "post-1" || got[0].AuthorID != "author-1" {
		t.Fatalf("expected one canonical candidate, got %#v", got)
	}
}

func TestRankHonorsSensitivePreferenceAndFutureTimestamps(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	candidates := []Candidate{
		{ID: "sensitive", AuthorID: "author-a", Visible: true, Sensitive: true, CreatedAt: now},
		{ID: "future", AuthorID: "author-b", Visible: true, CreatedAt: now.Add(24 * time.Hour)},
	}
	hidden := Rank(Request{Now: now, Limit: 10, AllowSensitive: false, Candidates: candidates})
	if len(hidden) != 1 || hidden[0].ID != "future" {
		t.Fatalf("sensitive candidate should be excluded, got %#v", hidden)
	}
	allowed := Rank(Request{Now: now, Limit: 10, AllowSensitive: true, Candidates: candidates})
	if len(allowed) != 2 {
		t.Fatalf("expected sensitive candidate when explicitly allowed, got %#v", allowed)
	}
	// A future timestamp is treated as age zero, never as a score bonus.
	for _, item := range allowed {
		if item.ID == "future" && item.Score > 0.3000001 {
			t.Fatalf("future timestamp received an unexpected recency bonus: %#v", item)
		}
	}
}

func TestRankNeverReturnsNonFiniteScores(t *testing.T) {
	now := time.Now().UTC()
	got := Rank(Request{Now: now, Candidates: []Candidate{{
		ID: "candidate", AuthorID: "author", Visible: true, CreatedAt: now,
		Features: Features{MutualAffinity: math.NaN(), InterestAffinity: math.Inf(1), Quality: math.Inf(-1)},
	}}})
	if len(got) != 1 || math.IsNaN(got[0].Score) || math.IsInf(got[0].Score, 0) || got[0].Score < 0 || got[0].Score > 1 {
		t.Fatalf("ranker emitted a non-finite or out-of-range score: %#v", got)
	}
}

func TestRankFailsClosedOnNegativeReportRisk(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	got := Rank(Request{Now: now, Candidates: []Candidate{
		{ID: "safe", AuthorID: "author-safe", Visible: true, CreatedAt: now, Features: Features{ReportRisk: 0.1}},
		{ID: "negative-risk", AuthorID: "author-negative", Visible: true, CreatedAt: now, Features: Features{ReportRisk: -0.01}},
	}})
	if len(got) != 1 || got[0].ID != "safe" {
		t.Fatalf("negative report-risk values must fail closed, got %#v", got)
	}
}

func TestRankFailsClosedOnNonFiniteReportRisk(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	got := Rank(Request{Now: now, Candidates: []Candidate{
		{ID: "safe", AuthorID: "author-safe", Visible: true, CreatedAt: now, Features: Features{ReportRisk: 0.1}},
		{ID: "nan-risk", AuthorID: "author-nan", Visible: true, CreatedAt: now, Features: Features{ReportRisk: math.NaN()}},
		{ID: "positive-infinite-risk", AuthorID: "author-inf", Visible: true, CreatedAt: now, Features: Features{ReportRisk: math.Inf(1)}},
		{ID: "negative-infinite-risk", AuthorID: "author-neg-inf", Visible: true, CreatedAt: now, Features: Features{ReportRisk: math.Inf(-1)}},
	}})
	if len(got) != 1 || got[0].ID != "safe" {
		t.Fatalf("non-finite report-risk candidates must fail closed, got %#v", got)
	}
}

func TestFeedModeProfilesFavorDifferentSignals(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	candidates := []Candidate{
		{ID: "followed", AuthorID: "author-a", Visible: true, CreatedAt: now, Features: Features{Following: true, InterestAffinity: 0.1, Quality: 0.5, Conversation: 0.2}},
		{ID: "discovery", AuthorID: "author-b", Visible: true, CreatedAt: now, Features: Features{InterestAffinity: 0.9, Quality: 0.9, Conversation: 0.2}},
	}
	following := Rank(Request{Now: now, FeedMode: "following", Limit: 2, Candidates: candidates})
	discover := Rank(Request{Now: now, FeedMode: "discover", Limit: 2, Candidates: candidates})
	if len(following) != 2 || following[0].ID != "followed" {
		t.Fatalf("following lane should prioritize followed-account signal: %#v", following)
	}
	if len(discover) != 2 || discover[0].ID != "discovery" {
		t.Fatalf("discover lane should prioritize interest and quality: %#v", discover)
	}
}

func TestRankDefaultsFeedModeAndRejectsUnsupportedModes(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	candidate := Candidate{ID: "post", AuthorID: "author", Visible: true, CreatedAt: now}
	defaultMode := Rank(Request{Now: now, Limit: 1, Candidates: []Candidate{candidate}})
	forYou := Rank(Request{Now: now, FeedMode: "for_you", Limit: 1, Candidates: []Candidate{candidate}})
	if len(defaultMode) != 1 || len(forYou) != 1 || defaultMode[0].Score != forYou[0].Score {
		t.Fatalf("empty feed mode must default to for_you: default=%#v for_you=%#v", defaultMode, forYou)
	}
	if got := Rank(Request{Now: now, FeedMode: "latest", Limit: 1, Candidates: []Candidate{candidate}}); len(got) != 0 {
		t.Fatalf("latest must be handled by Backend chronological pagination, got %#v", got)
	}
}
