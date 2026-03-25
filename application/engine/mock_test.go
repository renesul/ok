package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/llm"
	"go.uber.org/zap"
)

// --- Mock LLM Server ---

type llmResponses struct {
	decide  string // JSON content for Decide response
	plan    string // JSON content for CreatePlanStreaming (streamed)
	reflect string // JSON content for Reflect response
	summary string // content for ChatCompletionSync (pruning/summarization)
}

func newMockLLMServer(r *llmResponses) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		var parsed map[string]interface{}
		json.Unmarshal(body, &parsed)

		stream, _ := parsed["stream"].(bool)
		temp, _ := parsed["temperature"].(float64)
		maxTok, _ := parsed["max_tokens"].(float64)

		var content string
		switch {
		case stream && temp == 0.2 && maxTok == 800:
			// CreatePlanStreaming
			content = r.plan
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n",
				mustJSON(map[string]interface{}{
					"choices": []map[string]interface{}{
						{"delta": map[string]string{"content": content}},
					},
				}))
			return
		case !stream && temp == 0.2 && maxTok == 500:
			content = r.decide
		case !stream && temp == 0.1 && maxTok == 300:
			content = r.reflect
		default:
			// ChatCompletionSync fallback (pruning, summarization)
			content = r.summary
			if content == "" {
				content = "resumo"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": content}},
			},
		})
	}))
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// --- Mock Planner ---

type mockPlanner struct {
	tools     map[string]domain.Tool
	planErr   error
	planCount int
}

func newMockPlanner() *mockPlanner {
	return &mockPlanner{tools: make(map[string]domain.Tool)}
}

func (p *mockPlanner) Plan(decision domain.Decision, ctx *domain.AgentContext) (domain.Plan, error) {
	p.planCount++
	if p.planErr != nil {
		return domain.Plan{}, p.planErr
	}
	tool, ok := p.tools[decision.Tool]
	if !ok {
		return domain.Plan{}, fmt.Errorf("tool %q not found", decision.Tool)
	}
	return domain.Plan{Tool: tool, Input: decision.Input}, nil
}

func (p *mockPlanner) RegisterTool(tool domain.Tool) {
	p.tools[tool.Name()] = tool
}

func (p *mockPlanner) ToolDescriptions() string { return "mock tools" }
func (p *mockPlanner) Tools() map[string]domain.Tool {
	return p.tools
}

// --- Mock Executor ---

type mockExecutor struct {
	result    string
	err       error
	execCount int
	lastTool  string
}

func newMockExecutor(result string) *mockExecutor {
	return &mockExecutor{result: result}
}

func (e *mockExecutor) Execute(plan domain.Plan) (string, error) {
	e.execCount++
	e.lastTool = plan.Tool.Name()
	return e.result, e.err
}

// --- Mock Tool ---

type mockTool struct {
	name string
}

func (t *mockTool) Name() string        { return t.name }
func (t *mockTool) Description() string { return "mock " + t.name }
func (t *mockTool) Run(input string) (string, error) {
	return "mock-result: " + input, nil
}

// --- Test Engine Builder ---

func newTestEngine(serverURL string, planner domain.Planner, executor domain.Executor, limits domain.AgentLimits) *AgentEngine {
	llmClient := llm.NewClient(zap.NewNop())

	cfg := llm.ClientConfig{
		BaseURL:          serverURL,
		APIKey:           "test-key",
		Model:            "test-model",
		MaxContextTokens: 128000,
	}

	return NewAgentEngine(
		nil, // db — nil skips save
		llmClient,
		cfg,
		cfg, // llmFast = same server
		planner,
		executor,
		nil, // memory — nil skips memory operations
		nil, // execRepo — nil skips execution record
		limits,
		func() string { return "system prompt" },
		zap.NewNop(),
	)
}

func defaultLimits() domain.AgentLimits {
	return domain.AgentLimits{
		MaxSteps:    6,
		MaxAttempts: 4,
		TimeoutMs:   int(30 * time.Second / time.Millisecond),
	}
}

// newStatefulLLMServer — like newMockLLMServer but with a custom reflectFn for per-call responses.
func newStatefulLLMServer(resp *llmResponses, reflectFn func() string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		var parsed map[string]interface{}
		json.Unmarshal(body, &parsed)

		stream, _ := parsed["stream"].(bool)
		temp, _ := parsed["temperature"].(float64)
		maxTok, _ := parsed["max_tokens"].(float64)

		var content string
		switch {
		case stream && temp == 0.2 && maxTok == 800:
			content = resp.plan
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n",
				mustJSON(map[string]interface{}{
					"choices": []map[string]interface{}{
						{"delta": map[string]string{"content": content}},
					},
				}))
			return
		case !stream && temp == 0.2 && maxTok == 500:
			content = resp.decide
		case !stream && temp == 0.1 && maxTok == 300:
			content = reflectFn()
		default:
			content = resp.summary
			if content == "" {
				content = "resumo"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": content}},
			},
		})
	}))
}
