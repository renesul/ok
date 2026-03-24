package application

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/embedding"
	"go.uber.org/zap"
)

type EmbeddingService struct {
	client                 *embedding.Client
	messageRepository      domain.MessageRepository
	conversationRepository domain.ConversationRepository
	log                    *zap.Logger
}

func NewEmbeddingService(
	client *embedding.Client,
	messageRepository domain.MessageRepository,
	conversationRepository domain.ConversationRepository,
	log *zap.Logger,
) *EmbeddingService {
	return &EmbeddingService{
		client:                 client,
		messageRepository:      messageRepository,
		conversationRepository: conversationRepository,
		log:                    log.Named("service.embedding"),
	}
}

func (s *EmbeddingService) Enabled() bool {
	return s.client.Enabled()
}

func (s *EmbeddingService) EmbedAndStore(ctx context.Context, messages []domain.Message) {
	if !s.Enabled() || len(messages) == 0 {
		return
	}

	texts := make([]string, 0, len(messages))
	validMessages := make([]domain.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Content != "" && msg.ID != 0 {
			texts = append(texts, msg.Content)
			validMessages = append(validMessages, msg)
		}
	}

	if len(texts) == 0 {
		return
	}

	embeddings, err := s.client.Embed(ctx, texts)
	if err != nil {
		s.log.Debug("embed failed", zap.Error(err))
		return
	}

	for i, emb := range embeddings {
		if i >= len(validMessages) {
			break
		}
		msg := validMessages[i]
		if err := s.messageRepository.SaveEmbedding(ctx, msg.ID, msg.ConversationID, emb); err != nil {
			s.log.Debug("save embedding failed", zap.Uint("message_id", msg.ID), zap.Error(err))
		}
	}

	s.log.Debug("embeddings stored", zap.Int("count", len(embeddings)))
}

func (s *EmbeddingService) SearchSemantic(ctx context.Context, query string) ([]domain.Conversation, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("embedding not configured")
	}

	s.log.Debug("semantic search", zap.String("query", query))

	queryEmbeddings, err := s.client.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(queryEmbeddings) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	queryVec := queryEmbeddings[0]

	allEmbeddings, err := s.messageRepository.FindAllEmbeddings(ctx)
	if err != nil {
		return nil, fmt.Errorf("find embeddings: %w", err)
	}

	if len(allEmbeddings) == 0 {
		return []domain.Conversation{}, nil
	}

	convScores := make(map[uint]float32)
	for _, emb := range allEmbeddings {
		score := cosineSimilarity(queryVec, emb.Embedding)
		if score > convScores[emb.ConversationID] {
			convScores[emb.ConversationID] = score
		}
	}

	type scored struct {
		convID uint
		score  float32
	}
	var ranked []scored
	for convID, score := range convScores {
		if score > 0.3 {
			ranked = append(ranked, scored{convID, score})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > 20 {
		ranked = ranked[:20]
	}

	if len(ranked) == 0 {
		return []domain.Conversation{}, nil
	}

	var conversations []domain.Conversation
	for _, r := range ranked {
		conv, err := s.conversationRepository.FindByID(ctx, r.convID)
		if err != nil || conv == nil {
			continue
		}
		conversations = append(conversations, *conv)
	}

	return conversations, nil
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	denom := float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))
	if denom == 0 {
		return 0
	}
	return dot / denom
}
