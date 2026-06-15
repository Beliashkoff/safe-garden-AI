package admin

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// This file holds the deeper growth and answer-quality analytics that back the
// "Рост" and "Качество ответов" admin pages. All read-only over existing tables.

// --- Growth analytics ---

// Activation is the proxy activation funnel for a sign-up window.
type Activation struct {
	Signups     int64
	Activated   int64   // asked at least one question
	MedianHours float64 // median sign-up -> first question
}

// GetActivation returns the activation proxy for the last `days`.
func (s *Service) GetActivation(ctx context.Context, days int) (Activation, error) {
	r, err := s.store.ActivationSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return Activation{}, fmt.Errorf("admin: activation: %w", err)
	}
	return Activation{
		Signups:     r.Signups,
		Activated:   r.Activated,
		MedianHours: r.MedianSeconds / 3600,
	}, nil
}

// RetentionCohort is one weekly sign-up cohort with bucketed return rates.
// Eligible is the matured denominator; Retained is who actually came back.
type RetentionCohort struct {
	Week        time.Time
	Size        int64
	D1Eligible  int64
	D1Retained  int64
	D7Eligible  int64
	D7Retained  int64
	D30Eligible int64
	D30Retained int64
}

// GetRetention returns weekly cohorts over the last `days` (default 12 weeks).
func (s *Service) GetRetention(ctx context.Context, days int) ([]RetentionCohort, error) {
	rows, err := s.store.RetentionCohorts(ctx, s.sinceDays(days, 84))
	if err != nil {
		return nil, fmt.Errorf("admin: retention cohorts: %w", err)
	}
	out := make([]RetentionCohort, 0, len(rows))
	for _, r := range rows {
		out = append(out, RetentionCohort{
			Week:        r.Week.Time,
			Size:        r.Size,
			D1Eligible:  r.D1Eligible,
			D1Retained:  r.D1Retained,
			D7Eligible:  r.D7Eligible,
			D7Retained:  r.D7Retained,
			D30Eligible: r.D30Eligible,
			D30Retained: r.D30Retained,
		})
	}
	return out, nil
}

// DepthDistribution buckets users by question count plus avg/median/p90.
type DepthDistribution struct {
	Bucket1    int64
	Bucket2To4 int64
	Bucket5To9 int64
	Bucket10   int64
	Avg        float64
	Median     float64
	P90        float64
}

// GetConversationDepth returns the per-user question-count distribution.
func (s *Service) GetConversationDepth(ctx context.Context, days int) (DepthDistribution, error) {
	r, err := s.store.ConversationDepthSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return DepthDistribution{}, fmt.Errorf("admin: conversation depth: %w", err)
	}
	return DepthDistribution{
		Bucket1:    r.Bucket1,
		Bucket2To4: r.Bucket24,
		Bucket5To9: r.Bucket59,
		Bucket10:   r.Bucket10,
		Avg:        r.AvgMsgs,
		Median:     r.MedianMsgs,
		P90:        r.P90Msgs,
	}, nil
}

// HeatCell is one Moscow day-of-week/hour bucket. DOW 0=Sun..6=Sat.
type HeatCell struct {
	DOW   int32
	Hour  int32
	Count int64
}

// GetActivityHeatmap returns question volume by Moscow weekday and hour.
func (s *Service) GetActivityHeatmap(ctx context.Context, days int) ([]HeatCell, error) {
	rows, err := s.store.ActivityHeatmapSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return nil, fmt.Errorf("admin: activity heatmap: %w", err)
	}
	out := make([]HeatCell, 0, len(rows))
	for _, r := range rows {
		out = append(out, HeatCell{DOW: r.Dow, Hour: r.Hour, Count: r.Count})
	}
	return out, nil
}

// WeekPoint is one bucket of the weekly seasonal curve.
type WeekPoint struct {
	Week        time.Time
	Messages    int64
	ActiveUsers int64
}

// GetActivityByWeek returns the weekly seasonal curve (default ~6 months).
func (s *Service) GetActivityByWeek(ctx context.Context, days int) ([]WeekPoint, error) {
	rows, err := s.store.ActivityByWeekSince(ctx, s.sinceDays(days, 180))
	if err != nil {
		return nil, fmt.Errorf("admin: activity by week: %w", err)
	}
	out := make([]WeekPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, WeekPoint{Week: r.Week.Time, Messages: r.Messages, ActiveUsers: r.ActiveUsers})
	}
	return out, nil
}

// --- Answer quality ---

// CTRStat is impressions vs taps for one catalog slug.
type CTRStat struct {
	Slug        string
	Impressions int64
	Taps        int64
}

// CTROverview is the fertilizer-card click-through summary. Cards is the number
// of card blocks shown; Impressions is product-level (the CTR denominator).
type CTROverview struct {
	Cards       int64
	Impressions int64
	Taps        int64
	PerSlug     []CTRStat
}

// GetCTR returns the recommendation click-through stats for the last `days`.
func (s *Service) GetCTR(ctx context.Context, days int) (CTROverview, error) {
	since := s.sinceDays(days, 30)
	var o CTROverview

	cards, err := s.store.CardImpressionsSince(ctx, since)
	if err != nil {
		return o, fmt.Errorf("admin: card impressions: %w", err)
	}
	o.Cards = cards

	imps, err := s.store.CardImpressionsBySlugSince(ctx, since)
	if err != nil {
		return o, fmt.Errorf("admin: impressions by slug: %w", err)
	}
	taps, err := s.store.TapsBySlugSince(ctx, since)
	if err != nil {
		return o, fmt.Errorf("admin: taps by slug: %w", err)
	}

	byslug := map[string]*CTRStat{}
	for _, r := range imps {
		byslug[r.Slug] = &CTRStat{Slug: r.Slug, Impressions: r.Impressions}
		o.Impressions += r.Impressions
	}
	for _, r := range taps {
		if e, ok := byslug[r.Slug]; ok {
			e.Taps = r.Taps
		} else {
			byslug[r.Slug] = &CTRStat{Slug: r.Slug, Taps: r.Taps}
		}
		o.Taps += r.Taps
	}
	o.PerSlug = make([]CTRStat, 0, len(byslug))
	for _, v := range byslug {
		o.PerSlug = append(o.PerSlug, *v)
	}
	sort.Slice(o.PerSlug, func(i, j int) bool {
		if o.PerSlug[i].Impressions != o.PerSlug[j].Impressions {
			return o.PerSlug[i].Impressions > o.PerSlug[j].Impressions
		}
		return o.PerSlug[i].Taps > o.PerSlug[j].Taps
	})
	return o, nil
}

// LengthByVerdict is average answer length (tokens_out) split by feedback verdict.
type LengthByVerdict struct {
	AvgUp   float64
	AvgDown float64
	AvgNone float64
	NUp     int64
	NDown   int64
	NNone   int64
}

// GetLengthByVerdict returns answer-length-vs-satisfaction for the last `days`.
func (s *Service) GetLengthByVerdict(ctx context.Context, days int) (LengthByVerdict, error) {
	r, err := s.store.ResponseLengthByVerdictSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return LengthByVerdict{}, fmt.Errorf("admin: length by verdict: %w", err)
	}
	return LengthByVerdict{
		AvgUp:   r.AvgUp,
		AvgDown: r.AvgDown,
		AvgNone: r.AvgNone,
		NUp:     r.NUp,
		NDown:   r.NDown,
		NNone:   r.NNone,
	}, nil
}

// NegativeConversation is a chat accumulating dislikes. Conversation is masked.
type NegativeConversation struct {
	Conversation string
	Downs        int64
	LastDown     time.Time
}

// GetNegativeConversations lists chats with at least minDowns dislikes in `days`.
func (s *Service) GetNegativeConversations(ctx context.Context, days, minDowns int) ([]NegativeConversation, error) {
	if minDowns < 1 {
		minDowns = 2
	}
	rows, err := s.store.ConversationsWithNegativeFeedbackSince(ctx, db.ConversationsWithNegativeFeedbackSinceParams{
		Since:    s.sinceDays(days, 30),
		MinDowns: int64(minDowns),
	})
	if err != nil {
		return nil, fmt.Errorf("admin: negative conversations: %w", err)
	}
	out := make([]NegativeConversation, 0, len(rows))
	for _, r := range rows {
		out = append(out, NegativeConversation{
			Conversation: fmt.Sprintf("%x", r.ConversationID[:4]),
			Downs:        r.Downs,
			LastDown:     r.LastDown.Time,
		})
	}
	return out, nil
}

// FollowupRate is the share of answers immediately re-questioned (proxy for
// implicit dissatisfaction).
type FollowupRate struct {
	Answers   int64
	Followups int64
}

// GetFollowupRate returns the re-ask rate for the last `days`.
func (s *Service) GetFollowupRate(ctx context.Context, days int) (FollowupRate, error) {
	r, err := s.store.FollowupRateSince(ctx, s.sinceDays(days, 30))
	if err != nil {
		return FollowupRate{}, fmt.Errorf("admin: followup rate: %w", err)
	}
	return FollowupRate{Answers: r.Answers, Followups: r.Followups}, nil
}

// DownvotedMessage is one disliked answer with its question/answer text for
// operator review. The caller must never log this content (CLAUDE.md invariant #3).
type DownvotedMessage struct {
	Conversation string
	Question     string
	Answer       string
	HadCard      bool
	CreatedAt    time.Time
}

// ListDownvoted returns the newest disliked answers for the review queue.
func (s *Service) ListDownvoted(ctx context.Context, limit, offset int32) ([]DownvotedMessage, error) {
	limit = clamp(limit, 1, 100)
	if offset < 0 {
		offset = 0
	}
	rows, err := s.store.ListDownvotedMessages(ctx, db.ListDownvotedMessagesParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("admin: list downvoted: %w", err)
	}
	out := make([]DownvotedMessage, 0, len(rows))
	for _, r := range rows {
		out = append(out, DownvotedMessage{
			Conversation: fmt.Sprintf("%x", r.ConversationID[:4]),
			Question:     r.QuestionText.String,
			Answer:       r.AnswerText.String,
			HadCard:      r.HadCard,
			CreatedAt:    r.UpdatedAt.Time,
		})
	}
	return out, nil
}
