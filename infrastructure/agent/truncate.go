package agent

import "unicode/utf8"

// TruncateUTF8 trunca string em maxBytes sem corromper caracteres UTF-8
func TruncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	truncated := s[:maxBytes]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

// TruncateWithEllipsis trunca e adiciona "..." como indicador
func TruncateWithEllipsis(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return TruncateUTF8(s, maxBytes) + "..."
}
