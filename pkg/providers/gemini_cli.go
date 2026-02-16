package providers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

type GeminiCLIProvider struct {
	config config.GeminiCLIConfig
}

func NewGeminiCLIProvider(cfg config.GeminiCLIConfig) *GeminiCLIProvider {
	return &GeminiCLIProvider{
		config: cfg,
	}
}

func (p *GeminiCLIProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	// Find the last user message
	var lastUserMsg string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMsg = messages[i].Content
			break
		}
	}

	if lastUserMsg == "" {
		return &LLMResponse{
			Content:      "No user message found to execute.",
			FinishReason: "stop",
		}, nil
	}

	args := []string{"-p", lastUserMsg}

	// Handle Session
	// Gemini CLI currently only supports "latest" or numeric index.
	// We default to "latest" to maintain continuity for the single user.
	if p.config.ResumeSession {
		args = append(args, "--resume", "latest")
	}

	if p.config.YoloMode {
		args = append(args, "--yolo")
	}

	cmd := exec.CommandContext(ctx, p.config.BinaryPath, args...)
	if p.config.WorkingDir != "" {
		cmd.Dir = p.config.WorkingDir
	}

	output, err := cmd.CombinedOutput()
	outputStr := filterOutput(string(output))

	if err != nil {
		// If exit code is non-zero, it might still have useful output (stderr)
		return nil, fmt.Errorf("gemini cli execution failed: %v, output: %s", err, outputStr)
	}

	return &LLMResponse{
		Content:      outputStr,
		FinishReason: "stop",
	}, nil
}

func filterOutput(output string) string {
	lines := strings.Split(output, "\n")
	var filtered []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Filter out noisy system logs
		if strings.HasPrefix(trimmed, "YOLO mode is enabled") ||
			strings.HasPrefix(trimmed, "Loaded cached credentials") ||
			strings.HasPrefix(trimmed, "Hook registry initialized") ||
			strings.HasPrefix(trimmed, "Attempt ") && strings.Contains(trimmed, "failed") ||
			strings.Contains(trimmed, "pgrep: command not found") {
			continue
		}

		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n")
}

func (p *GeminiCLIProvider) GetDefaultModel() string {
	return "gemini-cli"
}
