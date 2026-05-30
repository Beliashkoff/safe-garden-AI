package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// loadHistory returns the last ~20 turns as neutral llm.Messages, chronological.
func (s *Service) loadHistory(ctx context.Context, convID uuid.UUID) ([]llm.Message, error) {
	rows, err := s.store.ListRecentMessages(ctx, db.ListRecentMessagesParams{
		ConversationID: convID, Limit: historyLimit,
	})
	if err != nil {
		return nil, err
	}
	reverseMessages(rows) // chronological for the model
	blocksByMsg, err := s.blocksFor(ctx, rows)
	if err != nil {
		return nil, err
	}
	return s.buildLLMHistory(ctx, rows, blocksByMsg), nil
}

// GetConversation returns the conversation id + the most recent page of messages
// (chronological), with a cursor for older messages.
func (s *Service) GetConversation(ctx context.Context, userID uuid.UUID, limit int) (ConversationView, error) {
	conv, err := s.store.GetOrCreateConversation(ctx, userID)
	if err != nil {
		return ConversationView{}, fmt.Errorf("chat: conversation: %w", err)
	}
	n := clampLimit(limit)
	rows, err := s.store.ListRecentMessages(ctx, db.ListRecentMessagesParams{
		ConversationID: conv.ID, Limit: n + 1,
	})
	if err != nil {
		return ConversationView{}, fmt.Errorf("chat: list messages: %w", err)
	}
	page, next := paginate(rows, n)
	views, err := s.viewsFor(ctx, page)
	if err != nil {
		return ConversationView{}, err
	}
	return ConversationView{ConversationID: conv.ID.String(), Messages: views, NextCursor: next}, nil
}

// ListMessages returns an older keyset page (GET /v1/conversation/messages).
func (s *Service) ListMessages(ctx context.Context, userID uuid.UUID, cursor string, limit int) (MessagesPage, error) {
	conv, err := s.store.GetOrCreateConversation(ctx, userID)
	if err != nil {
		return MessagesPage{}, fmt.Errorf("chat: conversation: %w", err)
	}
	beforeAt, beforeID, err := decodeCursor(cursor)
	if err != nil {
		return MessagesPage{}, err
	}
	n := clampLimit(limit)
	rows, err := s.store.ListMessagesBefore(ctx, db.ListMessagesBeforeParams{
		ConversationID:  conv.ID,
		BeforeCreatedAt: tsVal(beforeAt),
		BeforeID:        beforeID,
		Limit:           n + 1,
	})
	if err != nil {
		return MessagesPage{}, fmt.Errorf("chat: list messages: %w", err)
	}
	page, next := paginate(rows, n)
	views, err := s.viewsFor(ctx, page)
	if err != nil {
		return MessagesPage{}, err
	}
	return MessagesPage{Messages: views, NextCursor: next}, nil
}

// DeleteMessage removes the caller's own message (blocks cascade).
func (s *Service) DeleteMessage(ctx context.Context, userID, messageID uuid.UUID) error {
	rows, err := s.store.DeleteMessage(ctx, db.DeleteMessageParams{ID: messageID, UserID: userID})
	if err != nil {
		return fmt.Errorf("chat: delete message: %w", err)
	}
	if rows == 0 {
		return ErrMessageNotFound
	}
	return nil
}

// paginate takes newest-first rows (fetched with limit+1), trims to the page,
// and returns the page (still newest-first) plus the cursor to fetch older.
func paginate(rows []db.Message, limit int32) (page []db.Message, nextCursor string) {
	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}
	if hasMore && len(rows) > 0 {
		oldest := rows[len(rows)-1]
		nextCursor = encodeCursor(oldest.CreatedAt.Time, oldest.ID)
	}
	return rows, nextCursor
}

// viewsFor loads blocks for the rows and returns MessageViews in chronological
// order (rows arrive newest-first and are reversed here).
func (s *Service) viewsFor(ctx context.Context, rows []db.Message) ([]MessageView, error) {
	reverseMessages(rows)
	blocksByMsg, err := s.blocksFor(ctx, rows)
	if err != nil {
		return nil, err
	}
	views := make([]MessageView, 0, len(rows))
	for _, m := range rows {
		views = append(views, toMessageView(m, blocksByMsg[m.ID]))
	}
	return views, nil
}

func (s *Service) blocksFor(ctx context.Context, msgs []db.Message) (map[uuid.UUID][]db.MessageBlock, error) {
	if len(msgs) == 0 {
		return map[uuid.UUID][]db.MessageBlock{}, nil
	}
	ids := make([]uuid.UUID, len(msgs))
	for i, m := range msgs {
		ids[i] = m.ID
	}
	blocks, err := s.store.ListBlocksByMessageIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("chat: load blocks: %w", err)
	}
	byMsg := make(map[uuid.UUID][]db.MessageBlock, len(msgs))
	for _, b := range blocks {
		byMsg[b.MessageID] = append(byMsg[b.MessageID], b)
	}
	return byMsg, nil
}

func reverseMessages(m []db.Message) {
	for i, j := 0, len(m)-1; i < j; i, j = i+1, j-1 {
		m[i], m[j] = m[j], m[i]
	}
}
