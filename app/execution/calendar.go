package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ok/internal/config"
	"ok/internal/logger"
)

// CalendarTool provides Google Calendar and Microsoft Outlook event management.
type CalendarTool struct {
	cfg    config.CalendarConfig
	client *http.Client
}

// NewCalendarTool creates a new CalendarTool with the given config.
func NewCalendarTool(cfg config.CalendarConfig) *CalendarTool {
	return &CalendarTool{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *CalendarTool) Name() string        { return "calendar" }
func (t *CalendarTool) Description() string {
	return "Manage calendar events (Google Calendar and/or Microsoft Outlook). Actions: list (upcoming events), create (new event), delete (remove event)."
}

func (t *CalendarTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"list", "create", "delete"},
				"description": "Action: 'list' shows upcoming events, 'create' adds an event, 'delete' removes an event",
			},
			"provider": map[string]any{
				"type":        "string",
				"enum":        []string{"google", "outlook"},
				"description": "Calendar provider (default: google if configured, else outlook)",
			},
			// list params
			"days": map[string]any{
				"type":        "integer",
				"description": "Number of days ahead to list events (default: 7)",
				"minimum":     1.0,
				"maximum":     90.0,
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of events to return (default: 20)",
				"minimum":     1.0,
				"maximum":     100.0,
			},
			// create params
			"title": map[string]any{
				"type":        "string",
				"description": "Event title/summary",
			},
			"start": map[string]any{
				"type":        "string",
				"description": "Start datetime in ISO 8601 format (e.g. 2026-03-15T10:00:00)",
			},
			"end": map[string]any{
				"type":        "string",
				"description": "End datetime in ISO 8601 format (e.g. 2026-03-15T11:00:00)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Event description/notes",
			},
			"location": map[string]any{
				"type":        "string",
				"description": "Event location",
			},
			"timezone": map[string]any{
				"type":        "string",
				"description": "Timezone (e.g. America/Sao_Paulo). Default: UTC",
			},
			// delete params
			"event_id": map[string]any{
				"type":        "string",
				"description": "Event ID to delete (from list action)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *CalendarTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)

	// Determine provider
	provider, _ := args["provider"].(string)
	if provider == "" {
		if t.cfg.GoogleEnabled {
			provider = "google"
		} else if t.cfg.OutlookEnabled {
			provider = "outlook"
		} else {
			return ErrorResult("no calendar provider enabled (configure google or outlook in integrations.calendar)")
		}
	}

	switch action {
	case "list":
		days := 7
		if d, ok := args["days"].(float64); ok && int(d) > 0 {
			days = int(d)
		}
		maxResults := 20
		if m, ok := args["max_results"].(float64); ok && int(m) > 0 {
			maxResults = int(m)
		}
		return t.listEvents(ctx, provider, days, maxResults)

	case "create":
		title, _ := args["title"].(string)
		start, _ := args["start"].(string)
		end, _ := args["end"].(string)
		desc, _ := args["description"].(string)
		location, _ := args["location"].(string)
		tz, _ := args["timezone"].(string)
		if title == "" {
			return ErrorResult("'title' is required for create action")
		}
		if start == "" {
			return ErrorResult("'start' is required for create action")
		}
		if end == "" {
			end = start // same time start/end → all-day-ish
		}
		if tz == "" {
			tz = "UTC"
		}
		return t.createEvent(ctx, provider, title, start, end, tz, desc, location)

	case "delete":
		eventID, _ := args["event_id"].(string)
		if eventID == "" {
			return ErrorResult("'event_id' is required for delete action")
		}
		return t.deleteEvent(ctx, provider, eventID)

	default:
		return ErrorResult(fmt.Sprintf("unknown action %q", action))
	}
}

// --- Google Calendar (REST API v3) ---

func (t *CalendarTool) googleCalendarID() string {
	if t.cfg.GoogleCalendarID != "" {
		return t.cfg.GoogleCalendarID
	}
	return "primary"
}

func (t *CalendarTool) googleRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := "https://www.googleapis.com/calendar/v3" + path
	if strings.Contains(path, "?") {
		url += "&key=" + t.cfg.GoogleAPIKey
	} else {
		url += "?key=" + t.cfg.GoogleAPIKey
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return t.client.Do(req)
}

func (t *CalendarTool) listEventsGoogle(ctx context.Context, days, maxResults int) *ToolResult {
	now := time.Now().UTC()
	timeMin := now.Format(time.RFC3339)
	timeMax := now.AddDate(0, 0, days).Format(time.RFC3339)
	calID := t.googleCalendarID()

	path := fmt.Sprintf("/calendars/%s/events?timeMin=%s&timeMax=%s&maxResults=%d&singleEvents=true&orderBy=startTime",
		calID, timeMin, timeMax, maxResults)

	resp, err := t.googleRequest(ctx, "GET", path, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Google Calendar request failed: %v", err))
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return ErrorResult(fmt.Sprintf("Google Calendar API error %d: %s", resp.StatusCode, string(data)))
	}

	var result struct {
		Items []struct {
			ID      string `json:"id"`
			Summary string `json:"summary"`
			Start   struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"end"`
			Description string `json:"description"`
			Location    string `json:"location"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse Google Calendar response: %v", err))
	}

	if len(result.Items) == 0 {
		return NewToolResult(fmt.Sprintf("No events in the next %d days.", days))
	}

	out, _ := json.MarshalIndent(result.Items, "", "  ")
	return NewToolResult(fmt.Sprintf("Google Calendar events (%d):\n%s", len(result.Items), string(out)))
}

func (t *CalendarTool) createEventGoogle(ctx context.Context, title, start, end, tz, desc, location string) *ToolResult {
	calID := t.googleCalendarID()
	payload := map[string]any{
		"summary":  title,
		"start":    map[string]string{"dateTime": start, "timeZone": tz},
		"end":      map[string]string{"dateTime": end, "timeZone": tz},
		"description": desc,
		"location":    location,
	}
	body, _ := json.Marshal(payload)

	resp, err := t.googleRequest(ctx, "POST", fmt.Sprintf("/calendars/%s/events", calID), strings.NewReader(string(body)))
	if err != nil {
		return ErrorResult(fmt.Sprintf("Google Calendar create failed: %v", err))
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return ErrorResult(fmt.Sprintf("Google Calendar API error %d: %s", resp.StatusCode, string(data)))
	}

	var event struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
		HTMLLink string `json:"htmlLink"`
	}
	json.Unmarshal(data, &event)

	logger.InfoCF("calendar", "Google Calendar event created", map[string]any{"id": event.ID, "title": title})
	return NewToolResult(fmt.Sprintf("Event created: %s (ID: %s)\nLink: %s", event.Summary, event.ID, event.HTMLLink))
}

func (t *CalendarTool) deleteEventGoogle(ctx context.Context, eventID string) *ToolResult {
	calID := t.googleCalendarID()
	resp, err := t.googleRequest(ctx, "DELETE", fmt.Sprintf("/calendars/%s/events/%s", calID, eventID), nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Google Calendar delete failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return NewToolResult(fmt.Sprintf("Event %s deleted successfully.", eventID))
	}
	data, _ := io.ReadAll(resp.Body)
	return ErrorResult(fmt.Sprintf("Google Calendar delete error %d: %s", resp.StatusCode, string(data)))
}

// --- Microsoft Outlook (Graph API) ---

func (t *CalendarTool) outlookRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := "https://graph.microsoft.com/v1.0" + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+t.cfg.OutlookAccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return t.client.Do(req)
}

func (t *CalendarTool) outlookCalendarID() string {
	if t.cfg.OutlookCalendarID != "" {
		return t.cfg.OutlookCalendarID
	}
	return ""
}

func (t *CalendarTool) outlookEventsPath(extra string) string {
	calID := t.outlookCalendarID()
	if calID != "" {
		return fmt.Sprintf("/me/calendars/%s/events%s", calID, extra)
	}
	return "/me/events" + extra
}

func (t *CalendarTool) listEventsOutlook(ctx context.Context, days, maxResults int) *ToolResult {
	now := time.Now().UTC()
	timeMin := now.Format(time.RFC3339)
	timeMax := now.AddDate(0, 0, days).Format(time.RFC3339)

	path := t.outlookEventsPath(fmt.Sprintf("?$filter=start/dateTime ge '%s' and end/dateTime le '%s'&$top=%d&$orderby=start/dateTime",
		timeMin, timeMax, maxResults))

	resp, err := t.outlookRequest(ctx, "GET", path, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Outlook request failed: %v", err))
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return ErrorResult(fmt.Sprintf("Outlook API error %d: %s", resp.StatusCode, string(data)))
	}

	var result struct {
		Value []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
			Start   struct {
				DateTime string `json:"dateTime"`
				TimeZone string `json:"timeZone"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
			} `json:"end"`
			BodyPreview string `json:"bodyPreview"`
			Location    struct {
				DisplayName string `json:"displayName"`
			} `json:"location"`
		} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse Outlook response: %v", err))
	}

	if len(result.Value) == 0 {
		return NewToolResult(fmt.Sprintf("No Outlook events in the next %d days.", days))
	}

	out, _ := json.MarshalIndent(result.Value, "", "  ")
	return NewToolResult(fmt.Sprintf("Outlook events (%d):\n%s", len(result.Value), string(out)))
}

func (t *CalendarTool) createEventOutlook(ctx context.Context, title, start, end, tz, desc, location string) *ToolResult {
	payload := map[string]any{
		"subject": title,
		"start": map[string]string{
			"dateTime": start,
			"timeZone": tz,
		},
		"end": map[string]string{
			"dateTime": end,
			"timeZone": tz,
		},
		"body": map[string]string{
			"contentType": "text",
			"content":     desc,
		},
		"location": map[string]string{
			"displayName": location,
		},
	}
	body, _ := json.Marshal(payload)

	resp, err := t.outlookRequest(ctx, "POST", t.outlookEventsPath(""), strings.NewReader(string(body)))
	if err != nil {
		return ErrorResult(fmt.Sprintf("Outlook create failed: %v", err))
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		return ErrorResult(fmt.Sprintf("Outlook API error %d: %s", resp.StatusCode, string(data)))
	}

	var event struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		WebLink string `json:"webLink"`
	}
	json.Unmarshal(data, &event)

	logger.InfoCF("calendar", "Outlook event created", map[string]any{"id": event.ID, "title": title})
	return NewToolResult(fmt.Sprintf("Outlook event created: %s (ID: %s)", event.Subject, event.ID))
}

func (t *CalendarTool) deleteEventOutlook(ctx context.Context, eventID string) *ToolResult {
	resp, err := t.outlookRequest(ctx, "DELETE", fmt.Sprintf("/me/events/%s", eventID), nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Outlook delete failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return NewToolResult(fmt.Sprintf("Outlook event %s deleted successfully.", eventID))
	}
	data, _ := io.ReadAll(resp.Body)
	return ErrorResult(fmt.Sprintf("Outlook delete error %d: %s", resp.StatusCode, string(data)))
}

// --- Dispatch ---

func (t *CalendarTool) listEvents(ctx context.Context, provider string, days, maxResults int) *ToolResult {
	switch provider {
	case "google":
		return t.listEventsGoogle(ctx, days, maxResults)
	case "outlook":
		return t.listEventsOutlook(ctx, days, maxResults)
	default:
		return ErrorResult(fmt.Sprintf("unknown provider %q", provider))
	}
}

func (t *CalendarTool) createEvent(ctx context.Context, provider, title, start, end, tz, desc, location string) *ToolResult {
	switch provider {
	case "google":
		return t.createEventGoogle(ctx, title, start, end, tz, desc, location)
	case "outlook":
		return t.createEventOutlook(ctx, title, start, end, tz, desc, location)
	default:
		return ErrorResult(fmt.Sprintf("unknown provider %q", provider))
	}
}

func (t *CalendarTool) deleteEvent(ctx context.Context, provider, eventID string) *ToolResult {
	switch provider {
	case "google":
		return t.deleteEventGoogle(ctx, eventID)
	case "outlook":
		return t.deleteEventOutlook(ctx, eventID)
	default:
		return ErrorResult(fmt.Sprintf("unknown provider %q", provider))
	}
}
