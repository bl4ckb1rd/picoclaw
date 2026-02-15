package providers

import (
	"context"
	"fmt"
	"os/exec"

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
	outputStr := string(output)

	if err != nil {
		// If exit code is non-zero, it might still have useful output (stderr)
		return nil, fmt.Errorf("gemini cli execution failed: %v, output: %s", err, outputStr)
	}

	return &LLMResponse{
		Content:      outputStr,
		FinishReason: "stop",
	}, nil
}

func (p *GeminiCLIProvider) GetDefaultModel() string {
	return "gemini-cli"
}
