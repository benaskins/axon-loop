package loop_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	loop "github.com/benaskins/axon-loop"
	"github.com/benaskins/axon-talk/openai"
	tool "github.com/benaskins/axon-tool"
)

// TestLive_OrchestrationLoop reproduces code-lead's exact flow through axon-loop.
// Run with: OPENROUTER_API_KEY=sk-or-... go test -run TestLive_OrchestrationLoop -v -timeout 60s
func TestLive_OrchestrationLoop(t *testing.T) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY required")
	}

	model := "anthropic/claude-sonnet-4"
	client := openai.NewClient("https://openrouter.ai/api", apiKey)

	// code-lead's system prompt
	sysPrompt := `You are code-lead, a build orchestrator on the factory floor.

Your job is to execute a build plan step by step. For each step:

1. Call run_code_hand with the step title, description, and project directory.
   This invokes the coding agent to implement the step.
2. Call verify_build to check that the project compiles and tests pass.
   If verification fails, call run_code_hand again with feedback describing
   what went wrong. You get up to 3 attempts per step.
3. If verification passes, call read_diff to review the changes.
4. Call commit_step with the project directory and the commit message from
   the plan step.
5. Move to the next step.

Be methodical. Execute steps in order. Report what you're doing at each stage.
If a step fails after 3 attempts, report the failure and stop.

Do not skip steps. Do not modify the plan. Execute what is given to you.`

	userMsg := `Project directory: /tmp/test-project

Execute the following plan steps in order:

## Step 3: User authentication setup

Integrate axon-auth with email/password authentication. Create login and registration endpoints.

Commit message: ` + "`feat: implement user authentication with axon-auth`"

	// code-lead's 4 tools (stubbed to return immediately)
	tools := map[string]tool.ToolDef{
		"run_code_hand": {
			Name:        "run_code_hand",
			Description: "Invoke code-hand to implement a plan step.",
			Parameters: tool.ParameterSchema{
				Type: "object",
				Properties: map[string]tool.PropertySchema{
					"step_title":       {Type: "string", Description: "The step title"},
					"step_description": {Type: "string", Description: "Full step description"},
					"project_dir":      {Type: "string", Description: "Absolute path to the project directory"},
					"feedback":         {Type: "string", Description: "Optional feedback"},
				},
				Required: []string{"step_title", "step_description", "project_dir"},
			},
			Execute: func(tc *tool.ToolContext, args map[string]any) tool.ToolResult {
				t.Logf("TOOL CALLED: run_code_hand(%v)", args["step_title"])
				return tool.ToolResult{Content: "code-hand completed successfully. All files written."}
			},
		},
		"verify_build": {
			Name:        "verify_build",
			Description: "Run go build and go test in the project directory.",
			Parameters: tool.ParameterSchema{
				Type: "object",
				Properties: map[string]tool.PropertySchema{
					"project_dir": {Type: "string", Description: "Absolute path to the project directory"},
				},
				Required: []string{"project_dir"},
			},
			Execute: func(tc *tool.ToolContext, args map[string]any) tool.ToolResult {
				t.Logf("TOOL CALLED: verify_build")
				return tool.ToolResult{Content: "PASSED\nok test-project 0.5s"}
			},
		},
		"read_diff": {
			Name:        "read_diff",
			Description: "Read the git diff of uncommitted changes.",
			Parameters: tool.ParameterSchema{
				Type: "object",
				Properties: map[string]tool.PropertySchema{
					"project_dir": {Type: "string", Description: "Absolute path to the project directory"},
				},
				Required: []string{"project_dir"},
			},
			Execute: func(tc *tool.ToolContext, args map[string]any) tool.ToolResult {
				t.Logf("TOOL CALLED: read_diff")
				return tool.ToolResult{Content: "diff --git a/internal/auth/auth.go\n+package auth\n+func Login() {}"}
			},
		},
		"commit_step": {
			Name:        "commit_step",
			Description: "Stage all changes and commit with the given message.",
			Parameters: tool.ParameterSchema{
				Type: "object",
				Properties: map[string]tool.PropertySchema{
					"project_dir": {Type: "string", Description: "Absolute path to the project directory"},
					"message":     {Type: "string", Description: "Commit message"},
				},
				Required: []string{"project_dir", "message"},
			},
			Execute: func(tc *tool.ToolContext, args map[string]any) tool.ToolResult {
				t.Logf("TOOL CALLED: commit_step(%v)", args["message"])
				return tool.ToolResult{Content: "abc1234"}
			},
		},
	}

	think := false
	req := &loop.Request{
		Model: model,
		Messages: []loop.Message{
			{Role: loop.RoleSystem, Content: sysPrompt},
			{Role: loop.RoleUser, Content: userMsg},
		},
		MaxIterations: 20,
		Stream:        true,
		Think:         &think,
		Options:       map[string]any{"temperature": float64(0)},
	}

	start := time.Now()
	t.Logf("starting orchestration loop (stream=%v)...", req.Stream)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	result, err := loop.Run(ctx, loop.RunConfig{
		Client:  client,
		Request: req,
		Tools:   tools,
		Callbacks: loop.Callbacks{
			OnToken: func(token string) {
				fmt.Fprintf(os.Stderr, "%s", token)
			},
			OnToolUse: func(name string, args map[string]any) {
				t.Logf("[%s] OnToolUse: %s", time.Since(start).Round(time.Millisecond), name)
			},
		},
	})

	elapsed := time.Since(start)
	t.Logf("loop completed in %s", elapsed)

	if err != nil {
		t.Fatalf("loop error after %s: %v", elapsed, err)
	}

	t.Logf("result content: %q", result.Content)
}
