package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// Security & abuse analytics for the "Безопасность" admin page. Identifiers that
// are PII never leave the backend raw: user ids are masked to a hex prefix and
// emails to first-chars + domain (CLAUDE.md invariant #3). Callers must not log
// the results.

// minOtpRequesterCodes is the floor for the "top requesters" list, so a normal
// user with a couple of codes never appears — only flooders.
const minOtpRequesterCodes = 3

// maskEmail keeps the first chars of the local part plus the domain so the
// operator can spot a flooded mailbox / domain without seeing the full address.
func maskEmail(e string) string {
	at := strings.IndexByte(e, '@')
	if at <= 0 {
		return "***"
	}
	local, domain := e[:at], e[at:]
	keep := 2
	if len(local) < keep {
		keep = 1
	}
	return local[:keep] + "***" + domain
}

// SecurityEvent is one row of the user-facing security feed. User is masked.
type SecurityEvent struct {
	User      string
	Action    string
	IP        string
	CreatedAt time.Time
}

// ListSecurityEvents returns the newest high-signal security events.
func (s *Service) ListSecurityEvents(ctx context.Context, limit, offset int32) ([]SecurityEvent, error) {
	limit = clamp(limit, 1, 200)
	if offset < 0 {
		offset = 0
	}
	rows, err := s.store.ListSecurityEvents(ctx, db.ListSecurityEventsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("admin: list security events: %w", err)
	}
	out := make([]SecurityEvent, 0, len(rows))
	for _, r := range rows {
		ev := SecurityEvent{Action: r.Action, CreatedAt: r.CreatedAt.Time}
		if r.UserID.Valid {
			ev.User = fmt.Sprintf("%x", r.UserID.Bytes[:4])
		}
		if r.Ip != nil {
			ev.IP = r.Ip.String()
		}
		out = append(out, ev)
	}
	return out, nil
}

// OtpRequester is one email (masked) and how many codes it requested.
type OtpRequester struct {
	Email string
	Codes int64
}

// OtpStats is email-OTP delivery + abuse for a window.
type OtpStats struct {
	Issued        int64
	Used          int64
	Exhausted     int64
	ExpiredUnused int64
	DeliveryRate  float64
	TopRequesters []OtpRequester
}

// GetOtpStats returns OTP delivery/abuse stats for the last `days`.
func (s *Service) GetOtpStats(ctx context.Context, days int) (OtpStats, error) {
	since := s.sinceDays(days, 7)
	st, err := s.store.OtpStatsSince(ctx, since)
	if err != nil {
		return OtpStats{}, fmt.Errorf("admin: otp stats: %w", err)
	}
	o := OtpStats{
		Issued:        st.Issued,
		Used:          st.Used,
		Exhausted:     st.Exhausted,
		ExpiredUnused: st.ExpiredUnused,
	}
	if st.Issued > 0 {
		o.DeliveryRate = float64(st.Used) / float64(st.Issued)
	}
	reqs, err := s.store.TopOtpRequestersSince(ctx, db.TopOtpRequestersSinceParams{
		Since:    since,
		MinCodes: int64(minOtpRequesterCodes),
	})
	if err != nil {
		return o, fmt.Errorf("admin: top otp requesters: %w", err)
	}
	for _, r := range reqs {
		o.TopRequesters = append(o.TopRequesters, OtpRequester{Email: maskEmail(r.Email), Codes: r.Codes})
	}
	return o, nil
}

// SuspiciousIP is failed-auth volume from one IP, by source.
type SuspiciousIP struct {
	Source string
	IP     string
	Count  int64
}

// GetSuspiciousIPs returns IPs ranked by failed-auth volume for the last `days`.
func (s *Service) GetSuspiciousIPs(ctx context.Context, days int) ([]SuspiciousIP, error) {
	rows, err := s.store.SuspiciousIPsSince(ctx, s.sinceDays(days, 7))
	if err != nil {
		return nil, fmt.Errorf("admin: suspicious ips: %w", err)
	}
	out := make([]SuspiciousIP, 0, len(rows))
	for _, r := range rows {
		ip := ""
		if r.Ip != nil {
			ip = r.Ip.String()
		}
		out = append(out, SuspiciousIP{Source: r.Source, IP: ip, Count: r.Count})
	}
	return out, nil
}
