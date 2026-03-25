package domain

import "testing"

func TestAgentContext_IncrementAndLimit(t *testing.T) {
	ctx := NewAgentContext(3)

	if ctx.LimitReached() {
		t.Error("limit should not be reached at start")
	}

	ctx.Increment()
	ctx.Increment()
	if ctx.LimitReached() {
		t.Error("limit should not be reached at 2/3")
	}

	ctx.Increment()
	if !ctx.LimitReached() {
		t.Error("limit should be reached at 3/3")
	}
}

func TestAgentContext_AddAndString(t *testing.T) {
	ctx := NewAgentContext(10)
	ctx.Add("first")
	ctx.Add("second")
	ctx.Add("third")

	got := ctx.String()
	want := "first\nsecond\nthird"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
