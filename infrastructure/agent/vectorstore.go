package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	chromem "github.com/philippgille/chromem-go"
	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/embedding"
	"go.uber.org/zap"
)

const collectionName = "memory"

type VectorStore struct {
	db         *chromem.DB
	collection *chromem.Collection
	log        *zap.Logger
}

func NewVectorStore(dataDir string, embedClient *embedding.Client, log *zap.Logger) (*VectorStore, error) {
	dbPath := filepath.Join(dataDir, "chromem")
	logger := log.Named("vectorstore")

	embedFunc := chromem.EmbeddingFunc(func(ctx context.Context, text string) ([]float32, error) {
		vectors, err := embedClient.Embed(ctx, []string{text})
		if err != nil || len(vectors) == 0 {
			return nil, fmt.Errorf("embed: %w", err)
		}
		return vectors[0], nil
	})

	db, err := chromem.NewPersistentDB(dbPath, false)
	if err != nil {
		return nil, fmt.Errorf("open vectorstore: %w", err)
	}

	collection, err := db.GetOrCreateCollection(collectionName, nil, embedFunc)
	if err != nil {
		return nil, fmt.Errorf("create collection: %w", err)
	}

	logger.Debug("vectorstore ready", zap.String("path", dbPath))

	return &VectorStore{
		db:         db,
		collection: collection,
		log:        logger,
	}, nil
}

func (vs *VectorStore) Save(id, content, category string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metadata := map[string]string{"category": category}

	err := vs.collection.AddDocument(ctx, chromem.Document{
		ID:       id,
		Content:  content,
		Metadata: metadata,
	})
	if err != nil {
		vs.log.Debug("vectorstore save failed", zap.String("id", id), zap.Error(err))
	}
}

func (vs *VectorStore) Search(query string, limit int) ([]domain.MemoryEntry, error) {
	if limit <= 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := vs.collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("vectorstore search: %w", err)
	}

	entries := make([]domain.MemoryEntry, len(results))
	for i, r := range results {
		entries[i] = domain.MemoryEntry{
			ID:       r.ID,
			Content:  r.Content,
			Category: r.Metadata["category"],
		}
	}

	vs.log.Debug("vectorstore search", zap.String("query", query), zap.Int("results", len(entries)))
	return entries, nil
}

func (vs *VectorStore) Delete(ids ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := vs.collection.Delete(ctx, nil, nil, ids...); err != nil {
		vs.log.Debug("vectorstore delete failed", zap.Error(err))
	}
}
