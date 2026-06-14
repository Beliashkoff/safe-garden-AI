package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// SetFeedback records (or clears) the caller's thumbs up/down on one of their
// assistant messages (PUT /v1/messages/:id/feedback). Owner-scoped: the message
// must exist, belong to userID, and be an assistant turn, else ErrMessageNotFound
// (so a foreign or non-assistant id is indistinguishable from a missing one). An
// empty/"none"/"null" value clears the verdict; any other non-verdict is
// ErrInvalidFeedback. The whole operation is idempotent.
func (s *Service) SetFeedback(ctx context.Context, userID, messageID uuid.UUID, value string) error {
	verdict, clear, err := normalizeFeedback(value)
	if err != nil {
		return err
	}

	msg, err := s.store.GetMessageByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMessageNotFound
		}
		return fmt.Errorf("chat: set feedback: %w", err)
	}
	if msg.UserID != userID || msg.Role != "assistant" {
		return ErrMessageNotFound
	}

	if clear {
		if err := s.store.DeleteMessageFeedback(ctx, db.DeleteMessageFeedbackParams{
			MessageID: messageID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("chat: set feedback: %w", err)
		}
		return nil
	}
	if err := s.store.UpsertMessageFeedback(ctx, db.UpsertMessageFeedbackParams{
		MessageID: messageID, UserID: userID, Value: verdict,
	}); err != nil {
		return fmt.Errorf("chat: set feedback: %w", err)
	}
	return nil
}

// normalizeFeedback maps a raw value to a stored verdict ("up"/"down") or a clear
// flag. Empty, "none" and "null" clear; anything else but the two verdicts is
// ErrInvalidFeedback.
func normalizeFeedback(value string) (verdict string, clear bool, err error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "up":
		return "up", false, nil
	case "down":
		return "down", false, nil
	case "", "none", "null":
		return "", true, nil
	default:
		return "", false, ErrInvalidFeedback
	}
}
