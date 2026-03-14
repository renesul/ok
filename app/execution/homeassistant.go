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

// HomeAssistantTool provides control over Home Assistant via its REST API.
type HomeAssistantTool struct {
	cfg    config.HomeAssistantConfig
	client *http.Client
}

// NewHomeAssistantTool creates a new HomeAssistantTool with the given config.
func NewHomeAssistantTool(cfg config.HomeAssistantConfig) *HomeAssistantTool {
	return &HomeAssistantTool{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *HomeAssistantTool) Name() string        { return "home_assistant" }
func (t *HomeAssistantTool) Description() string {
	return "Control Home Assistant: get entity states, call services (turn on/off lights, switches, etc.), and list all entities."
}

func (t *HomeAssistantTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"get_state", "list_states", "call_service", "get_history"},
				"description": "Action: 'get_state' returns a single entity's state, 'list_states' lists all entities (optionally filtered), 'call_service' calls a HA service, 'get_history' returns state history",
			},
			"entity_id": map[string]any{
				"type":        "string",
				"description": "Entity ID (e.g. 'light.living_room', 'switch.kitchen', 'sensor.temperature')",
			},
			"domain": map[string]any{
				"type":        "string",
				"description": "Entity domain for call_service or filtering list_states (e.g. 'light', 'switch', 'climate', 'media_player')",
			},
			"service": map[string]any{
				"type":        "string",
				"description": "Service name for call_service (e.g. 'turn_on', 'turn_off', 'toggle', 'set_temperature')",
			},
			"service_data": map[string]any{
				"type":        "object",
				"description": "Additional data for call_service (e.g. {\"brightness\": 255, \"color_temp\": 4000})",
			},
			"filter": map[string]any{
				"type":        "string",
				"description": "Filter prefix for list_states (e.g. 'light.' to list only lights)",
			},
			"hours": map[string]any{
				"type":        "integer",
				"description": "Number of hours of history to retrieve for get_history (default: 24)",
				"minimum":     1.0,
				"maximum":     168.0,
			},
		},
		"required": []string{"action"},
	}
}

func (t *HomeAssistantTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)

	switch action {
	case "get_state":
		entityID, _ := args["entity_id"].(string)
		if entityID == "" {
			return ErrorResult("'entity_id' is required for get_state")
		}
		return t.getState(ctx, entityID)

	case "list_states":
		filter, _ := args["filter"].(string)
		if filter == "" {
			filter, _ = args["domain"].(string)
			if filter != "" && !strings.HasSuffix(filter, ".") {
				filter += "."
			}
		}
		return t.listStates(ctx, filter)

	case "call_service":
		domain, _ := args["domain"].(string)
		service, _ := args["service"].(string)
		entityID, _ := args["entity_id"].(string)
		if domain == "" {
			return ErrorResult("'domain' is required for call_service (e.g. 'light', 'switch')")
		}
		if service == "" {
			return ErrorResult("'service' is required for call_service (e.g. 'turn_on', 'turn_off')")
		}
		var serviceData map[string]any
		if sd, ok := args["service_data"].(map[string]any); ok {
			serviceData = sd
		} else {
			serviceData = map[string]any{}
		}
		if entityID != "" {
			serviceData["entity_id"] = entityID
		}
		return t.callService(ctx, domain, service, serviceData)

	case "get_history":
		entityID, _ := args["entity_id"].(string)
		hours := 24
		if h, ok := args["hours"].(float64); ok && int(h) > 0 {
			hours = int(h)
		}
		return t.getHistory(ctx, entityID, hours)

	default:
		return ErrorResult(fmt.Sprintf("unknown action %q", action))
	}
}

func (t *HomeAssistantTool) haRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := strings.TrimRight(t.cfg.URL, "/") + "/api" + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+t.cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	return t.client.Do(req)
}

func (t *HomeAssistantTool) getState(ctx context.Context, entityID string) *ToolResult {
	resp, err := t.haRequest(ctx, "GET", "/states/"+entityID, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Home Assistant request failed: %v", err))
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 404 {
		return ErrorResult(fmt.Sprintf("Entity %q not found", entityID))
	}
	if resp.StatusCode != 200 {
		return ErrorResult(fmt.Sprintf("Home Assistant API error %d: %s", resp.StatusCode, string(data)))
	}

	var state struct {
		EntityID   string         `json:"entity_id"`
		State      string         `json:"state"`
		Attributes map[string]any `json:"attributes"`
		LastChanged string        `json:"last_changed"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse HA response: %v", err))
	}

	out, _ := json.MarshalIndent(state, "", "  ")
	return NewToolResult(fmt.Sprintf("State of %s:\n%s", entityID, string(out)))
}

func (t *HomeAssistantTool) listStates(ctx context.Context, filter string) *ToolResult {
	resp, err := t.haRequest(ctx, "GET", "/states", nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Home Assistant request failed: %v", err))
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return ErrorResult(fmt.Sprintf("Home Assistant API error %d: %s", resp.StatusCode, string(data)))
	}

	var states []struct {
		EntityID   string `json:"entity_id"`
		State      string `json:"state"`
		LastChanged string `json:"last_changed"`
	}
	if err := json.Unmarshal(data, &states); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse HA response: %v", err))
	}

	// Filter by prefix if requested
	if filter != "" {
		var filtered []struct {
			EntityID   string `json:"entity_id"`
			State      string `json:"state"`
			LastChanged string `json:"last_changed"`
		}
		for _, s := range states {
			if strings.HasPrefix(s.EntityID, filter) {
				filtered = append(filtered, s)
			}
		}
		states = filtered
	}

	out, _ := json.MarshalIndent(states, "", "  ")
	return NewToolResult(fmt.Sprintf("Home Assistant entities (%d):\n%s", len(states), string(out)))
}

func (t *HomeAssistantTool) callService(ctx context.Context, domain, service string, data map[string]any) *ToolResult {
	body, _ := json.Marshal(data)
	path := fmt.Sprintf("/services/%s/%s", domain, service)

	resp, err := t.haRequest(ctx, "POST", path, strings.NewReader(string(body)))
	if err != nil {
		return ErrorResult(fmt.Sprintf("Home Assistant request failed: %v", err))
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return ErrorResult(fmt.Sprintf("Home Assistant service error %d: %s", resp.StatusCode, string(respData)))
	}

	entityID, _ := data["entity_id"].(string)
	msg := fmt.Sprintf("Service %s.%s called successfully", domain, service)
	if entityID != "" {
		msg += fmt.Sprintf(" for %s", entityID)
	}

	logger.InfoCF("homeassistant", "Service called", map[string]any{
		"domain":    domain,
		"service":   service,
		"entity_id": entityID,
	})
	return NewToolResult(msg)
}

func (t *HomeAssistantTool) getHistory(ctx context.Context, entityID string, hours int) *ToolResult {
	since := time.Now().Add(-time.Duration(hours) * time.Hour).UTC().Format(time.RFC3339)
	path := fmt.Sprintf("/history/period/%s", since)
	if entityID != "" {
		path += "?filter_entity_id=" + entityID
	}

	resp, err := t.haRequest(ctx, "GET", path, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Home Assistant request failed: %v", err))
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return ErrorResult(fmt.Sprintf("Home Assistant API error %d: %s", resp.StatusCode, string(data)))
	}

	// History is a list of lists of state objects
	var history [][]struct {
		EntityID string `json:"entity_id"`
		State    string `json:"state"`
		LastChanged string `json:"last_changed"`
	}
	if err := json.Unmarshal(data, &history); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse HA history: %v", err))
	}

	out, _ := json.MarshalIndent(history, "", "  ")
	label := entityID
	if label == "" {
		label = "all entities"
	}
	return NewToolResult(fmt.Sprintf("History for %s (last %dh):\n%s", label, hours, string(out)))
}
