package context

import (
	"strings"
	"testing"

	"ok/providers"
)

func msg(role, content string) providers.Message {
	return providers.Message{Role: role, Content: content}
}

func assistantWithTools(toolIDs ...string) providers.Message {
	calls := make([]providers.ToolCall, len(toolIDs))
	for i, id := range toolIDs {
		calls[i] = providers.ToolCall{ID: id, Type: "function"}
	}
	return providers.Message{Role: "assistant", ToolCalls: calls}
}

func toolResult(id string) providers.Message {
	return providers.Message{Role: "tool", Content: "result", ToolCallID: id}
}

func TestSanitizeHistoryForProvider_EmptyHistory(t *testing.T) {
	result := sanitizeHistoryForProvider(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty, got %d messages", len(result))
	}

	result = sanitizeHistoryForProvider([]providers.Message{})
	if len(result) != 0 {
		t.Fatalf("expected empty, got %d messages", len(result))
	}
}

func TestSanitizeHistoryForProvider_SingleToolCall(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		assistantWithTools("A"),
		toolResult("A"),
		msg("assistant", "done"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_MultiToolCalls(t *testing.T) {
	history := []providers.Message{
		msg("user", "do two things"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		toolResult("B"),
		msg("assistant", "both done"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 5 {
		t.Fatalf("expected 5 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_AssistantToolCallAfterPlainAssistant(t *testing.T) {
	history := []providers.Message{
		msg("user", "hi"),
		msg("assistant", "thinking"),
		assistantWithTools("A"),
		toolResult("A"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant")
}

func TestSanitizeHistoryForProvider_OrphanedLeadingTool(t *testing.T) {
	history := []providers.Message{
		toolResult("A"),
		msg("user", "hello"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

func TestSanitizeHistoryForProvider_ToolAfterUserDropped(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		toolResult("A"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

func TestSanitizeHistoryForProvider_ToolAfterAssistantNoToolCalls(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		msg("assistant", "hi"),
		toolResult("A"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant")
}

func TestSanitizeHistoryForProvider_AssistantToolCallAtStart(t *testing.T) {
	history := []providers.Message{
		assistantWithTools("A"),
		toolResult("A"),
		msg("user", "hello"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user")
}

func TestSanitizeHistoryForProvider_MultiToolCallsThenNewRound(t *testing.T) {
	history := []providers.Message{
		msg("user", "do two things"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		toolResult("B"),
		msg("assistant", "done"),
		msg("user", "hi"),
		assistantWithTools("C"),
		toolResult("C"),
		msg("assistant", "done again"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 9 {
		t.Fatalf("expected 9 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "tool", "assistant", "user", "assistant", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_ConsecutiveMultiToolRounds(t *testing.T) {
	history := []providers.Message{
		msg("user", "start"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		toolResult("B"),
		assistantWithTools("C", "D"),
		toolResult("C"),
		toolResult("D"),
		msg("assistant", "all done"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 8 {
		t.Fatalf("expected 8 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "tool", "assistant", "tool", "tool", "assistant")
}

func TestSanitizeHistoryForProvider_PlainConversation(t *testing.T) {
	history := []providers.Message{
		msg("user", "hello"),
		msg("assistant", "hi"),
		msg("user", "how are you"),
		msg("assistant", "fine"),
	}

	result := sanitizeHistoryForProvider(history)
	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}
	assertRoles(t, result, "user", "assistant", "user", "assistant")
}

func roles(msgs []providers.Message) []string {
	r := make([]string, len(msgs))
	for i, m := range msgs {
		r[i] = m.Role
	}
	return r
}

func assertRoles(t *testing.T, msgs []providers.Message, expected ...string) {
	t.Helper()
	if len(msgs) != len(expected) {
		t.Fatalf("role count mismatch: got %v, want %v", roles(msgs), expected)
	}
	for i, exp := range expected {
		if msgs[i].Role != exp {
			t.Errorf("message[%d]: got role %q, want %q", i, msgs[i].Role, exp)
		}
	}
}

// TestSanitizeHistoryForProvider_IncompleteToolResults tests the forward validation
// that ensures assistant messages with tool_calls have ALL matching tool results.
// This fixes the DeepSeek error: "An assistant message with 'tool_calls' must be
// followed by tool messages responding to each 'tool_call_id'."
func TestSanitizeHistoryForProvider_IncompleteToolResults(t *testing.T) {
	// Assistant expects tool results for both A and B, but only A is present
	history := []providers.Message{
		msg("user", "do two things"),
		assistantWithTools("A", "B"),
		toolResult("A"),
		// toolResult("B") is missing - this would cause DeepSeek to fail
		msg("user", "next question"),
		msg("assistant", "answer"),
	}

	result := sanitizeHistoryForProvider(history)
	// The assistant message with incomplete tool results should be dropped,
	// along with its partial tool result. The remaining messages are:
	// user ("do two things"), user ("next question"), assistant ("answer")
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "user", "assistant")
}

// TestSanitizeHistoryForProvider_MissingAllToolResults tests the case where
// an assistant message has tool_calls but no tool results follow at all.
func TestSanitizeHistoryForProvider_MissingAllToolResults(t *testing.T) {
	history := []providers.Message{
		msg("user", "do something"),
		assistantWithTools("A"),
		// No tool results at all
		msg("user", "hello"),
		msg("assistant", "hi"),
	}

	result := sanitizeHistoryForProvider(history)
	// The assistant message with no tool results should be dropped.
	// Remaining: user ("do something"), user ("hello"), assistant ("hi")
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "user", "assistant")
}

// TestSanitizeHistoryForProvider_PartialToolResultsInMiddle tests that
// incomplete tool results in the middle of a conversation are properly handled.
func TestSanitizeHistoryForProvider_PartialToolResultsInMiddle(t *testing.T) {
	history := []providers.Message{
		msg("user", "first"),
		assistantWithTools("A"),
		toolResult("A"),
		msg("assistant", "done"),
		msg("user", "second"),
		assistantWithTools("B", "C"),
		toolResult("B"),
		// toolResult("C") is missing
		msg("user", "third"),
		assistantWithTools("D"),
		toolResult("D"),
		msg("assistant", "all done"),
	}

	result := sanitizeHistoryForProvider(history)
	// First round is complete (user, assistant+tools, tool, assistant),
	// second round is incomplete and dropped (assistant+tools, partial tool),
	// third round is complete (user, assistant+tools, tool, assistant).
	// Remaining: user, assistant, tool, assistant, user, user, assistant, tool, assistant
	if len(result) != 9 {
		t.Fatalf("expected 9 messages, got %d: %+v", len(result), roles(result))
	}
	assertRoles(t, result, "user", "assistant", "tool", "assistant", "user", "user", "assistant", "tool", "assistant")
}

// --- compactToolHistory tests ---

func assistantWithNamedTools(toolCalls ...providers.ToolCall) providers.Message {
	return providers.Message{Role: "assistant", Content: "", ToolCalls: toolCalls}
}

func namedToolCall(id, name, args string) providers.ToolCall {
	return providers.ToolCall{
		ID:   id,
		Name: name,
		Function: &providers.FunctionCall{
			Name:      name,
			Arguments: args,
		},
	}
}

func toolResultWithContent(id, content string) providers.Message {
	return providers.Message{Role: "tool", Content: content, ToolCallID: id}
}

func TestCompactToolHistory_ShortHistory(t *testing.T) {
	// History shorter than keepRecentMessages → no compaction
	history := []providers.Message{
		msg("user", "hello"),
		msg("assistant", "hi"),
	}
	result := compactToolHistory(history)
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}

func TestCompactToolHistory_CompactsOldToolPairs(t *testing.T) {
	// Build history with old tool pairs + recent messages
	history := make([]providers.Message, 0, 20)

	// Old tool pair (will be compacted)
	history = append(history,
		msg("user", "list files"),
		assistantWithNamedTools(namedToolCall("t1", "shell_exec", `{"command":"ls -la"}`)),
		toolResultWithContent("t1", "total 42\n-rw-r--r-- 1 user group  1234 file.txt\n-rw-r--r-- 1 user group  5678 data.csv"),
	)

	// Another old tool pair
	history = append(history,
		msg("user", "read file"),
		assistantWithNamedTools(namedToolCall("t2", "read_file", `{"path":"/tmp/test.txt"}`)),
		toolResultWithContent("t2", "This is a very long file content that goes on and on..."),
	)

	// Recent messages (keepRecentMessages = 10, so add 10)
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			history = append(history, msg("user", "recent message"))
		} else {
			history = append(history, msg("assistant", "recent reply"))
		}
	}

	result := compactToolHistory(history)

	// Old tool pairs should be compacted:
	// - user("list files") stays
	// - assistant+tool_call + tool_result → single assistant "[tool: shell_exec...]"
	// - user("read file") stays
	// - assistant+tool_call + tool_result → single assistant "[tool: read_file...]"
	// Then 10 recent messages
	// Total: 2 user + 2 compacted assistant + 10 recent = 14
	if len(result) != 14 {
		t.Fatalf("expected 14 messages, got %d", len(result))
	}

	// Check that compacted messages contain tool info
	if result[1].Role != "assistant" {
		t.Fatalf("expected compacted assistant, got %s", result[1].Role)
	}
	if !strings.Contains(result[1].Content, "shell_exec") {
		t.Fatalf("expected shell_exec in compacted content, got: %s", result[1].Content)
	}
	if !strings.Contains(result[1].Content, "[tool:") {
		t.Fatalf("expected [tool: prefix in compacted content, got: %s", result[1].Content)
	}

	// Tool result messages should be gone
	for _, m := range result[:4] {
		if m.Role == "tool" {
			t.Fatal("expected no tool messages in compacted region")
		}
	}
}

func TestCompactToolHistory_PreservesRecentToolPairs(t *testing.T) {
	// Recent tool pairs within keepRecentMessages should NOT be compacted
	history := make([]providers.Message, 0, 10)

	// These are ALL recent (< keepRecentMessages total)
	history = append(history,
		msg("user", "q1"),
		msg("assistant", "a1"),
		msg("user", "run something"),
		assistantWithNamedTools(namedToolCall("t1", "shell_exec", `{"command":"pwd"}`)),
		toolResultWithContent("t1", "/home/user"),
		msg("assistant", "done"),
		msg("user", "q2"),
		msg("assistant", "a2"),
	)

	result := compactToolHistory(history)
	// All 8 messages should be preserved (< keepRecentMessages)
	if len(result) != 8 {
		t.Fatalf("expected 8 messages, got %d", len(result))
	}
	assertRoles(t, result, "user", "assistant", "user", "assistant", "tool", "assistant", "user", "assistant")
}

func TestCompactToolHistory_PlainMessagesNotAffected(t *testing.T) {
	// Old non-tool messages should pass through unchanged
	history := make([]providers.Message, 0, 15)

	// Old plain messages
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			history = append(history, msg("user", "old question"))
		} else {
			history = append(history, msg("assistant", "old answer"))
		}
	}
	// Recent messages
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			history = append(history, msg("user", "recent"))
		} else {
			history = append(history, msg("assistant", "reply"))
		}
	}

	result := compactToolHistory(history)
	// No tool pairs → all messages preserved
	if len(result) != 16 {
		t.Fatalf("expected 16 messages, got %d", len(result))
	}
}

func TestCompactToolHistory_MultipleToolCallsInOneMessage(t *testing.T) {
	history := make([]providers.Message, 0, 15)

	// Old assistant with 2 tool calls
	history = append(history,
		msg("user", "do two things"),
		assistantWithNamedTools(
			namedToolCall("t1", "shell_exec", `{"command":"ls"}`),
			namedToolCall("t2", "read_file", `{"path":"x.txt"}`),
		),
		toolResultWithContent("t1", "file1.txt"),
		toolResultWithContent("t2", "file contents here"),
	)

	// Recent messages
	for i := 0; i < 10; i++ {
		history = append(history, msg("user", "msg"))
	}

	result := compactToolHistory(history)

	// Old region: user + compacted assistant (2 tool calls merged) = 2
	// Recent: 10
	if len(result) != 12 {
		t.Fatalf("expected 12, got %d", len(result))
	}

	// Compacted message should mention both tools
	compactedMsg := result[1]
	if !strings.Contains(compactedMsg.Content, "shell_exec") || !strings.Contains(compactedMsg.Content, "read_file") {
		t.Fatalf("expected both tool names, got: %s", compactedMsg.Content)
	}
}
