package ranking

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

var topicTagPattern = regexp.MustCompile("^[\\p{L}\\p{N}_]{1,64}$")

type TopicCandidate struct {
	Tag              string    `json:"tag"`
	PostsCount       int64     `json:"postsCount"`
	LastUsedAt       time.Time `json:"lastUsedAt"`
	InterestAffinity float64   `json:"interestAffinity"`
}

type TopicRequest struct {
	Now        time.Time        `json:"now"`
	Candidates []TopicCandidate `json:"candidates"`
	Limit      int              `json:"limit"`
}

type RankedTopic struct {
	Tag        string   `json:"tag"`
	PostsCount int64    `json:"postsCount"`
	Score      float64  `json:"score"`
	Reasons    []string `json:"reasons"`
}

type scoredTopic struct {
	item     RankedTopic
	lastUsed time.Time
}

func ValidTopicTag(tag string) bool {
	return topicTagPattern.MatchString(strings.TrimSpace(tag))
}

// RankTopics blends seven-day public-post volume, recency and the Backend's
// viewer-specific interest affinity. Candidate retrieval and visibility stay
// in Backend; this service only orders the supplied, already-authorized topics.
func RankTopics(in TopicRequest) []RankedTopic {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	seen := make(map[string]struct{}, len(in.Candidates))
	items := make([]scoredTopic, 0, min(len(in.Candidates), 200))
	maxCount := int64(1)
	for i, candidate := range in.Candidates {
		if i >= 200 {
			break
		}
		tag := strings.ToLower(strings.TrimSpace(candidate.Tag))
		if !ValidTopicTag(tag) || candidate.PostsCount < 1 || candidate.LastUsedAt.IsZero() ||
			math.IsNaN(candidate.InterestAffinity) || math.IsInf(candidate.InterestAffinity, 0) {
			continue
		}
		age := now.Sub(candidate.LastUsedAt)
		if age > 7*24*time.Hour || age < -5*time.Minute {
			continue
		}
		if candidate.PostsCount > maxCount {
			maxCount = candidate.PostsCount
		}
		if _, duplicate := seen[tag]; duplicate {
			continue
		}
		seen[tag] = struct{}{}

		if age < 0 {
			age = 0
		}
		recency := math.Exp(-math.Ln2 * age.Hours() / 24)
		volume := math.Log1p(float64(candidate.PostsCount)) / math.Log1p(float64(maxCount))
		affinity := math.Max(0, math.Min(1, candidate.InterestAffinity))
		score := 0.42*recency + 0.33*volume + 0.25*affinity
		score = math.Max(0, math.Min(1, score))

		reasons := make([]string, 0, 3)
		if recency >= 0.5 {
			reasons = append(reasons, "recent")
		}
		if volume >= 0.7 {
			reasons = append(reasons, "high-public-volume")
		}
		if affinity >= 0.6 {
			reasons = append(reasons, "matches-viewer-interests")
		}
		items = append(items, scoredTopic{
			item:     RankedTopic{Tag: tag, PostsCount: candidate.PostsCount, Score: score, Reasons: reasons},
			lastUsed: candidate.LastUsedAt,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].item.Score != items[j].item.Score {
			return items[i].item.Score > items[j].item.Score
		}
		if items[i].item.PostsCount != items[j].item.PostsCount {
			return items[i].item.PostsCount > items[j].item.PostsCount
		}
		if !items[i].lastUsed.Equal(items[j].lastUsed) {
			return items[i].lastUsed.After(items[j].lastUsed)
		}
		return items[i].item.Tag < items[j].item.Tag
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]RankedTopic, 0, len(items))
	for _, item := range items {
		out = append(out, item.item)
	}
	return out
}
