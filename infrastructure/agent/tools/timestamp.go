package tools

import (
	"fmt"
	"strconv"

	"github.com/renesul/ok/domain"
	"strings"
	"time"
)

type TimestampTool struct{}

func (t *TimestampTool) Name() string        { return "timestamp" }
func (t *TimestampTool) Description() string {
	return "date/time operations. Input: 'now' for current time, 'unix' for current unix timestamp, 'unix:1710000000' to convert unix to date, 'parse:2024-01-15' to parse a date string"
}
func (t *TimestampTool) Safety() domain.ToolSafety          { return domain.ToolSafe }

func (t *TimestampTool) Run(input string) (string, error) {
	if input == "" || input == "now" {
		return time.Now().Format(time.RFC3339), nil
	}

	if strings.HasPrefix(input, "unix:") {
		ts, err := strconv.ParseInt(strings.TrimPrefix(input, "unix:"), 10, 64)
		if err != nil {
			return "", fmt.Errorf("unix timestamp invalido: %w", err)
		}
		return time.Unix(ts, 0).Format(time.RFC3339), nil
	}

	if strings.HasPrefix(input, "parse:") {
		dateStr := strings.TrimPrefix(input, "parse:")
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
			"02/01/2006",
			"01/02/2006",
		} {
			t, err := time.Parse(layout, dateStr)
			if err == nil {
				return fmt.Sprintf("%s (unix: %d)", t.Format(time.RFC3339), t.Unix()), nil
			}
		}
		return "", fmt.Errorf("formato de data nao reconhecido: %s", dateStr)
	}

	if input == "unix" {
		return strconv.FormatInt(time.Now().Unix(), 10), nil
	}

	return time.Now().Format(time.RFC3339), nil
}
