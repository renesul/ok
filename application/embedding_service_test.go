package application

import (
	"math"
	"testing"
)

func TestCosineSimilarity_Identical(t *testing.T) {
	v := []float32{1, 2, 3}
	sim := cosineSimilarity(v, v)
	if math.Abs(float64(sim)-1.0) > 0.001 {
		t.Errorf("identical vectors: similarity = %f, want ~1.0", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	sim := cosineSimilarity(a, b)
	if math.Abs(float64(sim)) > 0.001 {
		t.Errorf("orthogonal vectors: similarity = %f, want ~0.0", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("different lengths: similarity = %f, want 0", sim)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("zero vector: similarity = %f, want 0", sim)
	}
}
