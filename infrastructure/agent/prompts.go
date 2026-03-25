package agent

import (
	"fmt"
	"strings"

	"github.com/renesul/ok/domain"
)

func BuildPlanningPrompt(goal string, toolDescriptions string, memories []string) string {
	var parts []string

	parts = append(parts, `You are an expert autonomous task planner. Given an objective, create a structured plan with 2 to 6 steps.

Each step MUST use one of the available tools. Decompose the objective into atomic and sequential actions.

GUIDE: Map the user's intent to the correct tool:
- search internet/documentation → web_search
- execute code → repl (respect language: JavaScript=node, Python=python)
- navigate/open site → browser
- search project files → search
- terminal commands / git / npm / tests → shell
- read file → file_read
- write new file → file_write
- edit file → file_edit
- fix bug → file_read + file_edit + shell
Respect EVERYTHING the user explicitly requested (language, format, tool, tone). NEVER substitute without asking.`)

	parts = append(parts, "Available tools:\n"+toolDescriptions)

	if len(memories) > 0 {
		parts = append(parts, "Relevant memories:\n"+strings.Join(memories, "\n"))
	}

	parts = append(parts, fmt.Sprintf(`Objective: %s

Respond ONLY with valid JSON in the exact format:
{"reasoning":"step-by-step deduction of the overall plan", "steps":[{"name":"short description","tool":"tool_name","input":"value","purpose":"why this step is needed"}]}

IMPORTANT: Use ONLY tools that exist in the list above. Each step must have name, tool, input, and purpose.`, goal))

	return strings.Join(parts, "\n\n")
}

func BuildReflectionPrompt(goal string, executedSteps []domain.PlannedStep, lastResult string) string {
	var stepsDesc []string
	for i, step := range executedSteps {
		status := step.Status
		if status == "" {
			status = "pending"
		}
		line := fmt.Sprintf("%d. [%s] %s (tool: %s)", i+1, status, step.Name, step.Tool)
		if step.Output != "" {
			output := TruncateWithEllipsis(step.Output, 300)
			line += "\n   Result: " + output
		}
		stepsDesc = append(stepsDesc, line)
	}

	return fmt.Sprintf(`You are an execution evaluator for an autonomous agent. Analyze the progress and decide the next step.

Objective: %s

Executed steps:
%s

Last result: %s

Analysis:
- Was the result valid and useful?
- Has the objective been achieved?
- Does the plan need adjustment?

Respond ONLY with valid JSON:
{"reason":"deep reasoning about the result and next steps", "action":"continue|replan|done|error", "final_answer":"final answer if action=done", "revised_plan":[{"name":"...","tool":"...","input":"...","purpose":"..."}]}

Actions:
- "continue": proceed to the next step of the plan
- "replan": replace remaining steps (provide revised_plan)
- "done": objective completely achieved (provide final_answer)
- "error": impossible to complete task (NEVER emit 'error' without attempting at least 3 totally distinct fallback strategies. Exhaust all resources before giving up. Provide the reason)`, goal, strings.Join(stepsDesc, "\n"), lastResult)
}
