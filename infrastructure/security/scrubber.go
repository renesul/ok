package security

import "regexp"

const redacted = "[REDACTED]"

type pattern struct {
	re          *regexp.Regexp
	replacement string
}

// SecretScrubber detecta e remove segredos de texto antes de enviar a LLMs externos ou persistir
type SecretScrubber struct {
	patterns []pattern
}

func NewSecretScrubber() *SecretScrubber {
	return &SecretScrubber{
		patterns: []pattern{
			// AWS Access Key
			{regexp.MustCompile(`AKIA[0-9A-Z]{16}`), redacted},

			// OpenAI API Keys
			{regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`), redacted},
			{regexp.MustCompile(`sk-proj-[a-zA-Z0-9_\-]{20,}`), redacted},

			// Private Keys (RSA, EC, DSA, OPENSSH)
			{regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----[\s\S]*?-----END (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`), redacted},

			// JWT tokens
			{regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}`), redacted},

			// Bearer tokens
			{regexp.MustCompile(`Bearer [a-zA-Z0-9\-._~+/]{20,}=*`), redacted},

			// Generic .env secrets (password=xxx, api_key=xxx, token=xxx, etc)
			{regexp.MustCompile(`(?i)(password|secret|token|api_key|apikey|auth_token|api_secret)\s*[:=]\s*["']?([^\s"']{8,})["']?`), "$1=" + redacted},

			// Hex secrets (40+ chars) associated with key-like names
			{regexp.MustCompile(`(?i)(secret|token|key|password)\s*[:=]\s*[0-9a-f]{40,}`), "$1=" + redacted},

			// GitHub tokens
			{regexp.MustCompile(`ghp_[a-zA-Z0-9]{36,}`), redacted},
			{regexp.MustCompile(`github_pat_[a-zA-Z0-9_]{22,}`), redacted},

			// Slack tokens
			{regexp.MustCompile(`xox[bpras]-[a-zA-Z0-9\-]{10,}`), redacted},

			// Generic long base64-like secrets (standalone, 40+ chars)
			{regexp.MustCompile(`(?i)(?:key|secret|token|password)\s*[:=]\s*["']([A-Za-z0-9+/=]{40,})["']`), redacted},
		},
	}
}

// Scrub aplica todos os patterns e substitui segredos encontrados
func (s *SecretScrubber) Scrub(text string) string {
	if s == nil || text == "" {
		return text
	}
	for _, p := range s.patterns {
		text = p.re.ReplaceAllString(text, p.replacement)
	}
	return text
}
