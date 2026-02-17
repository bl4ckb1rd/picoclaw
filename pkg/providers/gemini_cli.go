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
	// Find the system prompt and last user message
	var systemPrompt string
	var lastUserMsg string
	var history []string

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else if msg.Role == "user" {
			lastUserMsg = msg.Content
			history = append(history, "User: "+msg.Content)
		} else if msg.Role == "assistant" {
			history = append(history, "Assistant: "+msg.Content)
		} else if msg.Role == "tool" {
			history = append(history, "Tool Result: "+msg.Content)
		}
	}

	if lastUserMsg == "" {
		return &LLMResponse{
			Content:      "No user message found to execute.",
			FinishReason: "stop",
		}, nil
	}

	// Build a composite prompt for the CLI
	// We include the system prompt to ensure the CLI respects picoclaw rules/soul/agents.md
	// We only include the last few history items if necessary, but gemini-cli --resume latest
	// already handles some history. However, it doesn't know about picoclaw's system files.
	var compositePrompt strings.Builder
	if systemPrompt != "" {
		compositePrompt.WriteString(systemPrompt)
		compositePrompt.WriteString("\n\n---\n\n")
	}
	
	// Add recent context if it's a subagent or complex task
	// (Gemini CLI resume might handle main thread, but subagents are independent)
	compositePrompt.WriteString("Current Task: ")
	compositePrompt.WriteString(lastUserMsg)

	args := []string{"-p", compositePrompt.String()}

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
		if isNoise(line) {
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
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}

	// Filter out noisy system logs
	if strings.HasPrefix(trimmed, "YOLO mode is enabled") ||
		strings.HasPrefix(trimmed, "Loaded cached credentials") ||
		strings.HasPrefix(trimmed, "Hook registry initialized") ||
		(strings.HasPrefix(trimmed, "Attempt ") && strings.Contains(trimmed, "failed")) ||
		strings.Contains(trimmed, "pgrep: command not found") ||
		strings.HasPrefix(trimmed, "at ") || // Stack trace lines
		strings.Contains(trimmed, "file:///") || // File paths in stack traces
		strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "}") || // JSON blocks
		strings.HasPrefix(trimmed, "\"") || // JSON properties
		strings.Contains(trimmed, "RESOURCEEXHAUSTED") ||
		strings.Contains(trimmed, "MODELCAPACITYEXHAUSTED") ||
		strings.Contains(trimmed, "Too Many Requests") {
		return true
	}

	return false
}

func filterOutput(output string) string {
	lines := strings.Split(output, "\n")
	var filtered []string
	inJSON := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Detect JSON blocks to skip them entirely
		if trimmed == "{" || trimmed == "[" {
			inJSON = true
			continue
		}
		if inJSON {
			if trimmed == "}" || trimmed == "]" || trimmed == "}," || trimmed == "]," {
				inJSON = false
			}
			continue
		}

		if isNoise(line) {
			continue
		}

		filtered = append(filtered, line)
	}

	result := strings.Join(filtered, "\n")
	return strings.TrimSpace(result)
}

func (p *GeminiCLIProvider) GetDefaultModel() string {
	return "gemini-cli"
}
