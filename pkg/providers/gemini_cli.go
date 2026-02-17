package providers

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

type GeminiCLIProvider struct {
	config config.GeminiCLIConfig
	bus    *bus.MessageBus
}

func NewGeminiCLIProvider(cfg config.GeminiCLIConfig, bus *bus.MessageBus) *GeminiCLIProvider {
	return &GeminiCLIProvider{
		config: cfg,
		bus:    bus,
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

	// Determine model
	selectedModel := p.config.Model
	if model != "" && model != "gemini-cli" && model != "default" {
		selectedModel = model
	}

	if selectedModel != "" {
		args = append(args, "-m", selectedModel)
	}

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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	cmd.Stderr = cmd.Stdout // Merge stderr into stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start gemini cli: %v", err)
	}

	channel, _ := options["channel"].(string)
	chatID, _ := options["chat_id"].(string)

	var fullOutput strings.Builder
	var finalAnswer strings.Builder
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		line := scanner.Text()
		fullOutput.WriteString(line + "\n")

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Filter noise
		if isNoise(trimmed) {
			continue
		}

		// Stream thoughts in real-time
		if strings.HasPrefix(trimmed, "I will ") || strings.HasPrefix(trimmed, "I'll ") {
			if p.bus != nil && channel != "" && chatID != "" {
				p.bus.PublishOutbound(bus.OutboundMessage{
					Channel: channel,
					ChatID:  chatID,
					Content: "💭 " + trimmed,
				})
			}
			continue
		}

		finalAnswer.WriteString(line + "\n")
	}

	err = cmd.Wait()
	outputStr := strings.TrimSpace(finalAnswer.String())

	if err != nil {
		// If exit code is non-zero, return full output for debugging
		return nil, fmt.Errorf("gemini cli execution failed: %v, output: %s", err, fullOutput.String())
	}

	return &LLMResponse{
		Content:      outputStr,
		FinishReason: "stop",
	}, nil
}

func isNoise(line string) bool {
	return strings.HasPrefix(line, "YOLO mode is enabled") ||
		strings.HasPrefix(line, "Loaded cached credentials") ||
		strings.HasPrefix(line, "Hook registry initialized") ||
		(strings.HasPrefix(line, "Attempt ") && strings.Contains(line, "failed")) ||
		strings.Contains(line, "pgrep: command not found")
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
