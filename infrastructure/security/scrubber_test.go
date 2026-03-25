package security

import (
	"strings"
	"testing"
)

func TestScrubber_AWSKey(t *testing.T) {
	s := NewSecretScrubber()
	input := "key is AKIAIOSFODNN7EXAMPLE"
	result := s.Scrub(input)
	if strings.Contains(result, "AKIA") {
		t.Errorf("AWS key not scrubbed: %q", result)
	}
}

func TestScrubber_OpenAIKey(t *testing.T) {
	s := NewSecretScrubber()

	input := "api key: sk-proj-abc123def456ghi789jkl012mno345pqr678stu901vwx234yz567"
	result := s.Scrub(input)
	if strings.Contains(result, "sk-proj-") {
		t.Errorf("OpenAI project key not scrubbed: %q", result)
	}

	input2 := "key: sk-abcdef1234567890abcdef1234567890"
	result2 := s.Scrub(input2)
	if strings.Contains(result2, "sk-abcdef") {
		t.Errorf("OpenAI key not scrubbed: %q", result2)
	}
}

func TestScrubber_JWT(t *testing.T) {
	s := NewSecretScrubber()
	input := "token: eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	result := s.Scrub(input)
	if strings.Contains(result, "eyJ") {
		t.Errorf("JWT not scrubbed: %q", result)
	}
}

func TestScrubber_BearerToken(t *testing.T) {
	s := NewSecretScrubber()
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	result := s.Scrub(input)
	if strings.Contains(result, "Bearer eyJ") {
		t.Errorf("Bearer token not scrubbed: %q", result)
	}
}

func TestScrubber_EnvSecret(t *testing.T) {
	s := NewSecretScrubber()
	input := `password=mySuperSecret123!`
	result := s.Scrub(input)
	if strings.Contains(result, "mySuperSecret") {
		t.Errorf("env secret not scrubbed: %q", result)
	}
}

func TestScrubber_GitHubToken(t *testing.T) {
	s := NewSecretScrubber()
	input := "token: ghp_aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789"
	result := s.Scrub(input)
	if strings.Contains(result, "ghp_") {
		t.Errorf("GitHub token not scrubbed: %q", result)
	}
}

func TestScrubber_SafeText(t *testing.T) {
	s := NewSecretScrubber()
	input := "Hello, this is a normal message about programming in Go."
	result := s.Scrub(input)
	if result != input {
		t.Errorf("safe text was modified: %q", result)
	}
}

func TestScrubber_MultipleSecrets(t *testing.T) {
	s := NewSecretScrubber()
	input := "AWS: AKIAIOSFODNN7EXAMPLE and OpenAI: sk-abcdefghijklmnopqrstuvwxyz1234"
	result := s.Scrub(input)
	if strings.Contains(result, "AKIA") || strings.Contains(result, "sk-abcdef") {
		t.Errorf("not all secrets scrubbed: %q", result)
	}
}

func TestScrubber_NilScrubber(t *testing.T) {
	var s *SecretScrubber
	result := s.Scrub("test")
	if result != "test" {
		t.Errorf("nil scrubber should return original text, got %q", result)
	}
}
