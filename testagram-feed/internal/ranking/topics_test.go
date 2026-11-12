package ranking

import (
	"math"
	"testing"
	"time"
)

func TestRankTopicsUsesVolumeRecencyAndViewerAffinity(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	got := RankTopics(TopicRequest{
		Now: now,
		Limit: 10,
		Candidates: []TopicCandidate{
			{Tag: "generic", PostsCount: 1000, LastUsedAt: now.Add(-20 * time.Hour), InterestAffinity: 0.05},
			{Tag: "my-interest", PostsCount: 120, LastUsedAt: now.Add(-2 * time.Hour), InterestAffinity: 0.95},
			{Tag: "stale", PostsCount: 1000, LastUsedAt: now.Add(-6 * 24 * time.Hour), InterestAffinity: 0},
		},
	})
	if len(got) != 3 {
		t.Fatalf("got %d topics, want 3", len(got))
	}
	if got[0].Tag != "my-interest" {
		t.Fatalf("personalized topic ranked first = %q, want my-interest; all=%+v", got[0].Tag, got)
	}
	if got[0].Score <= got[1].Score {
		t.Fatalf("topic scores are not descending: %+v", got)
	}
	if len(got[0].Reasons) == 0 || got[0].Reasons[0] == "" {
		t.Fatalf("personalized ranking reasons missing: %+v", got[0])
	}
}

func TestRankTopicsDeduplicatesAndRejectsInvalidCandidates(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	got := RankTopics(TopicRequest{
		Now: now,
		Limit: 10,
		Candidates: []TopicCandidate{
			{Tag: "Testagram", PostsCount: 10, LastUsedAt: now, InterestAffinity: 0.5},
			{Tag: "testagram", PostsCount: 20, LastUsedAt: now, InterestAffinity: 0.8},
			{Tag: "bad tag", PostsCount: 50, LastUsedAt: now},
			{Tag: "empty", PostsCount: 0, LastUsedAt: now},
			{Tag: "NaN", PostsCount: 10, LastUsedAt: now, InterestAffinity: math.NaN()},
			{Tag: "東京", PostsCount: 5, LastUsedAt: now, InterestAffinity: 0.2},
		},
	})
	if len(got) != 2 {
		t.Fatalf("got %d valid unique topics, want 2: %+v", len(got), got)
	}
	if got[0].Tag != "testagram" || got[1].Tag != "東京" {
		t.Fatalf("unexpected normalized topic results: %+v", got)
	}
}

func TestRankTopicsExcludesOutsideSevenDayWindowAndFutureData(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	got := RankTopics(TopicRequest{
		Now: now,
		Limit: 10,
		Candidates: []TopicCandidate{
			{Tag: "current", PostsCount: 10, LastUsedAt: now.Add(-time.Hour), InterestAffinity: 0.5},
			{Tag: "stale", PostsCount: 100000, LastUsedAt: now.Add(-8 * 24 * time.Hour), InterestAffinity: 1},
			{Tag: "future-corrupt", PostsCount: 100000, LastUsedAt: now.Add(10 * time.Minute), InterestAffinity: 1},
		},
	})
	if len(got) != 1 || got[0].Tag != "current" {
		t.Fatalf("outside-window topics must be excluded, got %+v", got)
	}
}

func TestValidTopicTagAcceptsUnicodeAndRejectsUnsafeInput(t *testing.T) {
	for _, value := range []string{"testagram", "Kiswahili", "東京", "kenya_2026"} {
		if !ValidTopicTag(value) {
			t.Errorf("ValidTopicTag(%q) = false", value)
		}
	}
	for _, value := range []string{"", "bad tag", "bad/tag", "x!", string(make([]byte, 65))} {
		if ValidTopicTag(value) {
			t.Errorf("ValidTopicTag(%q) = true", value)
		}
	}
}
