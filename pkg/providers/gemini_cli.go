package providers

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

var imagePathRegex = regexp.MustCompile(`\[image: ([^\]]+)\]`)

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
	var imagePaths []string

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else if msg.Role == "user" {
			lastUserMsg = msg.Content
			// Extract image paths if present in the message
			matches := imagePathRegex.FindAllStringSubmatch(msg.Content, -1)
			for _, match := range matches {
				if len(match) > 1 {
					imagePaths = append(imagePaths, match[1])
				}
			}
		}
	}

	if lastUserMsg == "" {
		return &LLMResponse{
			Content:      "No user message found to execute.",
			FinishReason: "stop",
		}, nil
	}

	// Build a composite prompt for the CLI
	var compositePrompt strings.Builder
	if systemPrompt != "" {
		compositePrompt.WriteString(systemPrompt)
		compositePrompt.WriteString("\n\n---\n\n")
	}

	// Add image instructions to the prompt if any images were sent
	if len(imagePaths) > 0 {
		compositePrompt.WriteString("## Vision Context\n")
		compositePrompt.WriteString("The user has attached images to this request. They are available locally at the following paths. ")
		compositePrompt.WriteString("Please use your vision capabilities or tools to analyze them:\n")
		for _, path := range imagePaths {
			compositePrompt.WriteString(fmt.Sprintf("- %s\n", path))
		}
		compositePrompt.WriteString("\n---\n\n")
	}

	compositePrompt.WriteString("Current Task: ")
	compositePrompt.WriteString(lastUserMsg)

	// Determine initial model
	selectedModel := p.config.Model
	if model != "" && model != "gemini-cli" && model != "default" {
		selectedModel = model
	}

	// Retry logic parameters
	maxRetries := 3
	backoff := 2 * time.Second
	if b, ok := options["base_backoff"].(time.Duration); ok {
		backoff = b
	}

	// Tiered fallbacks for resilience
	fallbacks := []string{
		"gemini-3-flash-preview",
		"gemini-2.0-flash",
		"gemini-1.5-flash",
		"", // "auto" - let CLI decide
	}

	// Fix for trustedFolders.json error seen in logs
	// Ensure the config directory exists and the file is a valid JSON
	trustedFoldersPath := "/root/.gemini/trustedFolders.json"
	// We only try this if we are likely in the container (root user)
	if _, err := exec.LookPath("mkdir"); err == nil {
		exec.Command("mkdir", "-p", "/root/.gemini").Run()
		// If the file doesn't exist or is invalid, overwrite with empty object
		exec.Command("sh", "-c", "echo '{}' > "+trustedFoldersPath).Run()
	}

	var lastErr error
	var lastRawOutput string

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			// Select fallback for this attempt
			if i-1 < len(fallbacks) {
				fallback := fallbacks[i-1]
				if selectedModel != fallback {
					msg := fmt.Sprintf("⚠️ Quota exceeded for %s. Switching to %s...", selectedModel, func() string {
						if fallback == "" {
							return "auto (default)"
						}
						return fallback
					}())

					selectedModel = fallback
					if p.bus != nil {
						channel, _ := options["channel"].(string)
						chatID, _ := options["chat_id"].(string)
						if channel != "" && chatID != "" {
							p.bus.PublishOutbound(bus.OutboundMessage{
								Channel: channel,
								ChatID:  chatID,
								Content: msg,
							})
						}
					}
				}
			} else {
				// Standard wait if we already exhausted fallbacks or same model
				time.Sleep(backoff)
				backoff *= 2
			}
		}

		args := []string{"-p", compositePrompt.String()}
		if selectedModel != "" {
			args = append(args, "-m", selectedModel)
		}

		if p.config.ResumeSession {
			args = append(args, "--resume", "latest")
		}
		if p.config.YoloMode {
			args = append(args, "--yolo")
		}

		resp, rawOutput, err := p.runGeminiCommand(ctx, args, options)
		lastRawOutput = rawOutput
		if err == nil {
			return resp, nil
		}

		lastErr = err
		// Only retry on quota/capacity errors
		if !strings.Contains(rawOutput, "RESOURCEEXHAUSTED") &&
			!strings.Contains(rawOutput, "MODELCAPACITYEXHAUSTED") &&
			!strings.Contains(rawOutput, "QuotaError") &&
			!strings.Contains(rawOutput, "Too Many Requests") {
			break
		}
	}

	return nil, fmt.Errorf("gemini cli failed after retries: %v, output: %s", lastErr, lastRawOutput)
}

func (p *GeminiCLIProvider) runGeminiCommand(ctx context.Context, args []string, options map[string]interface{}) (*LLMResponse, string, error) {
	cmd := exec.CommandContext(ctx, p.config.BinaryPath, args...)
	if p.config.WorkingDir != "" {
		cmd.Dir = p.config.WorkingDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	cmd.Stderr = cmd.Stdout // Merge stderr into stdout

	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("failed to start gemini cli: %v", err)
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
					Metadata: map[string]string{
						"is_thought": "true",
					},
				})
			}
			continue
		}

		finalAnswer.WriteString(line + "\n")
	}

	err = cmd.Wait()
	rawOutput := fullOutput.String()
	outputStr := strings.TrimSpace(finalAnswer.String())

	// Log the raw output for debugging infrastructure/quota issues
	logger.DebugCF("gemini-cli", "Raw output captured", map[string]interface{}{
		"exit_code":   0,
		"output_len":  len(rawOutput),
		"raw_content": rawOutput,
	})

	if err != nil {
		return nil, rawOutput, err
	}

	return &LLMResponse{
		Content:      outputStr,
		FinishReason: "stop",
	}, rawOutput, nil
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
		strings.Contains(trimmed, "QuotaError") ||
		strings.Contains(trimmed, "retryDelayMs") ||
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
