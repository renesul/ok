package application

import (
	"testing"
)

func strPtr(s string) *string { return &s }

func floatPtr(f float64) *float64 { return &f }

func TestLinearizeMessages_SimpleChain(t *testing.T) {
	// A → B → C (linear chain)
	mapping := map[string]chatGPTNode{
		"a": {
			ID:       "a",
			Parent:   nil,
			Children: []string{"b"},
			Message: &chatGPTMessage{
				Author:  chatGPTAuthor{Role: "user"},
				Content: chatGPTContent{ContentType: "text", Parts: []any{"hello"}},
			},
		},
		"b": {
			ID:       "b",
			Parent:   strPtr("a"),
			Children: []string{"c"},
			Message: &chatGPTMessage{
				Author:  chatGPTAuthor{Role: "assistant"},
				Content: chatGPTContent{ContentType: "text", Parts: []any{"hi there"}},
			},
		},
		"c": {
			ID:       "c",
			Parent:   strPtr("b"),
			Children: []string{},
			Message: &chatGPTMessage{
				Author:  chatGPTAuthor{Role: "user"},
				Content: chatGPTContent{ContentType: "text", Parts: []any{"thanks"}},
			},
		},
	}

	msgs := linearizeMessages(mapping, 1700000000)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].content != "hello" {
		t.Errorf("msg[0] = %q, want 'hello'", msgs[0].content)
	}
	if msgs[1].content != "hi there" {
		t.Errorf("msg[1] = %q, want 'hi there'", msgs[1].content)
	}
	if msgs[2].content != "thanks" {
		t.Errorf("msg[2] = %q, want 'thanks'", msgs[2].content)
	}
}

func TestLinearizeMessages_Branching(t *testing.T) {
	// Root → child1 and child2 (follows last child = child2)
	mapping := map[string]chatGPTNode{
		"root": {
			ID:       "root",
			Parent:   nil,
			Children: []string{"child1", "child2"},
		},
		"child1": {
			ID:     "child1",
			Parent: strPtr("root"),
			Message: &chatGPTMessage{
				Author:  chatGPTAuthor{Role: "user"},
				Content: chatGPTContent{ContentType: "text", Parts: []any{"old branch"}},
			},
		},
		"child2": {
			ID:     "child2",
			Parent: strPtr("root"),
			Message: &chatGPTMessage{
				Author:  chatGPTAuthor{Role: "user"},
				Content: chatGPTContent{ContentType: "text", Parts: []any{"latest branch"}},
			},
		},
	}

	msgs := linearizeMessages(mapping, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (last branch), got %d", len(msgs))
	}
	if msgs[0].content != "latest branch" {
		t.Errorf("content = %q, want 'latest branch'", msgs[0].content)
	}
}

func TestLinearizeMessages_EmptyMapping(t *testing.T) {
	msgs := linearizeMessages(nil, 0)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for nil mapping, got %d", len(msgs))
	}

	msgs = linearizeMessages(map[string]chatGPTNode{}, 0)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for empty mapping, got %d", len(msgs))
	}
}

func TestLinearizeMessages_OrphanNodes(t *testing.T) {
	// Node with parent that doesn't exist in mapping → treated as root
	mapping := map[string]chatGPTNode{
		"orphan": {
			ID:     "orphan",
			Parent: strPtr("nonexistent"),
			Message: &chatGPTMessage{
				Author:  chatGPTAuthor{Role: "user"},
				Content: chatGPTContent{ContentType: "text", Parts: []any{"i am alone"}},
			},
		},
	}

	msgs := linearizeMessages(mapping, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message from orphan, got %d", len(msgs))
	}
	if msgs[0].content != "i am alone" {
		t.Errorf("content = %q, want 'i am alone'", msgs[0].content)
	}
}

func TestLinearizeMessages_NilContent(t *testing.T) {
	// Node with nil message → should be skipped without panic
	mapping := map[string]chatGPTNode{
		"root": {
			ID:       "root",
			Parent:   nil,
			Children: []string{"child"},
			Message:  nil, // no message
		},
		"child": {
			ID:     "child",
			Parent: strPtr("root"),
			Message: &chatGPTMessage{
				Author:  chatGPTAuthor{Role: "system"}, // system → filtered out
				Content: chatGPTContent{ContentType: "text", Parts: []any{"system msg"}},
			},
		},
	}

	msgs := linearizeMessages(mapping, 0)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages (nil + system filtered), got %d", len(msgs))
	}
}

func TestLinearizeMessages_FallbackTime(t *testing.T) {
	ts := 1700000000.5
	mapping := map[string]chatGPTNode{
		"a": {
			ID:     "a",
			Parent: nil,
			Message: &chatGPTMessage{
				Author:     chatGPTAuthor{Role: "user"},
				Content:    chatGPTContent{ContentType: "text", Parts: []any{"hi"}},
				CreateTime: &ts,
			},
		},
	}

	msgs := linearizeMessages(mapping, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].createdAt.Unix() != 1700000000 {
		t.Errorf("createdAt = %v, want unix 1700000000", msgs[0].createdAt)
	}
}
