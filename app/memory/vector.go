// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package memory

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ok/internal/utils"
)

// Interaction is a single stored interaction with its embedding.
type Interaction struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Channel   string    `json:"channel,omitempty"`
	SessionKey string   `json:"session_key,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// StoreMeta holds metadata about the interaction store.
type StoreMeta struct {
	Dimension      int    `json:"dimension"`
	Count          int    `json:"count"`
	EmbeddingModel string `json:"embedding_model"`
}

// InteractionStore persists interactions with their vector embeddings.
//
// Files:
//   - messages.json  — array of Interaction structs
//   - embeddings.bin — flat binary, float32 little-endian, dimension * count values
//   - metadata.json  — StoreMeta (dimension, count, model)
type InteractionStore struct {
	dir  string
	mu   sync.RWMutex
	meta StoreMeta

	// In-memory cache loaded on Open().
	interactions []Interaction
	embeddings   [][]float32
}

// NewInteractionStore creates a store in the given directory.
// Call Open() to load existing data from disk.
func NewInteractionStore(dir string) *InteractionStore {
	return &InteractionStore{dir: dir}
}

// Open loads existing data from disk. Safe to call if the directory is empty.
func (s *InteractionStore) Open() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}

	// Load metadata
	metaPath := filepath.Join(s.dir, "metadata.json")
	if data, err := os.ReadFile(metaPath); err == nil {
		json.Unmarshal(data, &s.meta)
	}

	// Load messages
	msgsPath := filepath.Join(s.dir, "messages.json")
	if data, err := os.ReadFile(msgsPath); err == nil {
		json.Unmarshal(data, &s.interactions)
	}

	// Load embeddings
	embPath := filepath.Join(s.dir, "embeddings.bin")
	if data, err := os.ReadFile(embPath); err == nil && s.meta.Dimension > 0 {
		s.embeddings = decodeEmbeddings(data, s.meta.Dimension)
	}

	// Consistency check
	if len(s.interactions) != len(s.embeddings) {
		// Truncate to the shorter length to recover from partial writes
		n := min(len(s.interactions), len(s.embeddings))
		s.interactions = s.interactions[:n]
		s.embeddings = s.embeddings[:n]
		s.meta.Count = n
	}

	return nil
}

// Add stores a new interaction with its embedding vector.
func (s *InteractionStore) Add(interaction Interaction, embedding []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set dimension from first embedding
	if s.meta.Dimension == 0 {
		s.meta.Dimension = len(embedding)
	}

	s.interactions = append(s.interactions, interaction)
	s.embeddings = append(s.embeddings, embedding)
	s.meta.Count = len(s.interactions)

	return s.flushLocked()
}

// Search returns the top-K most similar interactions to the query embedding.
func (s *InteractionStore) Search(queryEmb []float32, topK int, minSimilarity float64) []ScoredInteraction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.embeddings) == 0 {
		return nil
	}

	type scored struct {
		idx   int
		score float64
	}

	scores := make([]scored, 0, len(s.embeddings))
	for i, emb := range s.embeddings {
		sim := cosineSimilarity(queryEmb, emb)
		if sim >= minSimilarity {
			scores = append(scores, scored{idx: i, score: sim})
		}
	}

	// Simple selection sort for top-K (N is small for personal assistant)
	if topK > len(scores) {
		topK = len(scores)
	}
	for i := range topK {
		maxIdx := i
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[maxIdx].score {
				maxIdx = j
			}
		}
		scores[i], scores[maxIdx] = scores[maxIdx], scores[i]
	}

	results := make([]ScoredInteraction, topK)
	for i := range topK {
		results[i] = ScoredInteraction{
			Interaction: s.interactions[scores[i].idx],
			Score:       scores[i].score,
		}
	}
	return results
}

// Count returns the number of stored interactions.
func (s *InteractionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.interactions)
}

// SetEmbeddingModel records which model was used for embeddings.
func (s *InteractionStore) SetEmbeddingModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meta.EmbeddingModel = model
}

// ScoredInteraction pairs an interaction with its similarity score.
type ScoredInteraction struct {
	Interaction Interaction
	Score       float64
}

func (s *InteractionStore) flushLocked() error {
	// Write messages.json
	msgsData, err := json.Marshal(s.interactions)
	if err != nil {
		return err
	}
	if err := utils.WriteFileAtomic(filepath.Join(s.dir, "messages.json"), msgsData, 0o600); err != nil {
		return err
	}

	// Write embeddings.bin
	embData := encodeEmbeddings(s.embeddings)
	if err := utils.WriteFileAtomic(filepath.Join(s.dir, "embeddings.bin"), embData, 0o600); err != nil {
		return err
	}

	// Write metadata.json
	metaData, err := json.Marshal(s.meta)
	if err != nil {
		return err
	}
	return utils.WriteFileAtomic(filepath.Join(s.dir, "metadata.json"), metaData, 0o600)
}

// encodeEmbeddings serializes embeddings to flat binary (float32 little-endian).
func encodeEmbeddings(embeddings [][]float32) []byte {
	if len(embeddings) == 0 {
		return nil
	}
	dim := len(embeddings[0])
	buf := make([]byte, len(embeddings)*dim*4)
	offset := 0
	for _, emb := range embeddings {
		for _, v := range emb {
			binary.LittleEndian.PutUint32(buf[offset:], math.Float32bits(v))
			offset += 4
		}
	}
	return buf
}

// decodeEmbeddings deserializes flat binary to embedding vectors.
func decodeEmbeddings(data []byte, dimension int) [][]float32 {
	if dimension == 0 || len(data) < dimension*4 {
		return nil
	}
	count := len(data) / (dimension * 4)
	result := make([][]float32, count)
	offset := 0
	for i := range count {
		vec := make([]float32, dimension)
		for j := range dimension {
			vec[j] = math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))
			offset += 4
		}
		result[i] = vec
	}
	return result
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		normA += fa * fa
		normB += fb * fb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
