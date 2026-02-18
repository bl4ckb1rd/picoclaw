package providers

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// ... (existing tests)

func TestGeminiCLIProvider_Chat_Streaming(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "mock-gemini-stream-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `#!/bin/sh
echo "I will search for weather"
echo "I will execute curl"
echo "The weather is sunny"
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0755)

	msgBus := bus.NewMessageBus()
	cfg := config.GeminiCLIConfig{
		Enabled:    true,
		BinaryPath: tmpFile.Name(),
	}
	p := NewGeminiCLIProvider(cfg, msgBus)

	messages := []Message{
		{Role: "user", Content: "weather"},
	}

	// Channel to collect streamed thoughts
	type thoughtResult struct {
		content   string
		isThought bool
	}
	thoughts := make(chan thoughtResult, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			msg, ok := msgBus.SubscribeOutbound(ctx)
			if !ok {
				return
			}
			if strings.HasPrefix(msg.Content, "💭 ") {
				thoughts <- thoughtResult{
					content:   msg.Content,
					isThought: msg.Metadata != nil && msg.Metadata["is_thought"] == "true",
				}
			}
		}
	}()

	resp, err := p.Chat(ctx, messages, nil, "gemini-cli", map[string]interface{}{
		"channel": "test",
		"chat_id": "123",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify final response doesn't contain thoughts
	if strings.Contains(resp.Content, "I will") {
		t.Errorf("final response should not contain thoughts, got: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "The weather is sunny") {
		t.Errorf("expected final answer in response, got: %q", resp.Content)
	}

	// Verify thoughts were streamed
	expectedThoughts := []string{"💭 I will search for weather", "💭 I will execute curl"}
	for _, expected := range expectedThoughts {
		select {
		case got := <-thoughts:
			if got.content != expected {
				t.Errorf("expected thought %q, got %q", expected, got.content)
			}
			if !got.isThought {
				t.Errorf("expected is_thought metadata to be true for %q", expected)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("timed out waiting for thought: %q", expected)
		}
	}
}

func TestGeminiCLIProvider_Chat_NoUserMessage(t *testing.T) {
	cfg := config.GeminiCLIConfig{
		Enabled:    true,
		BinaryPath: "echo",
	}
	p := NewGeminiCLIProvider(cfg, nil)

	messages := []Message{
		{Role: "system", Content: "You are an assistant"},
	}

	resp, err := p.Chat(context.Background(), messages, nil, "gemini-cli", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "No user message found to execute."
	if resp.Content != expected {
		t.Errorf("expected %q, got %q", expected, resp.Content)
	}
}

// Test with a mock binary to verify command construction
func TestGeminiCLIProvider_Chat_CommandExecution(t *testing.T) {
	// Create a dummy script that acts as the gemini binary
	tmpFile, err := os.CreateTemp("", "mock-gemini-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `#!/bin/sh
echo "$*"
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0755)

	cfg := config.GeminiCLIConfig{
		Enabled:       true,
		BinaryPath:    tmpFile.Name(),
		ResumeSession: true,
		YoloMode:      true,
	}
	p := NewGeminiCLIProvider(cfg, nil)

	messages := []Message{
		{Role: "system", Content: "IDENTITY: BOT"},
		{Role: "user", Content: "hello gemini"},
	}

	resp, err := p.Chat(context.Background(), messages, nil, "gemini-cli", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(resp.Content, "IDENTITY: BOT") {
		t.Errorf("expected system prompt in output, got %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "Current Task: hello gemini") {
		t.Errorf("expected user message in output, got %q", resp.Content)
	}
}

func TestGeminiCLIProvider_Chat_ModelSelection(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "mock-gemini-model-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Script that finds the argument after -m
	content := `#!/bin/sh
while [ $# -gt 0 ]; do
  if [ "$1" = "-m" ]; then
    echo "Model used: $2"
    exit 0
  fi
  shift
done
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0755)

	cfg := config.GeminiCLIConfig{
		Enabled:    true,
		BinaryPath: tmpFile.Name(),
		Model:      "gemini-2.0-flash",
	}
	p := NewGeminiCLIProvider(cfg, nil)

	messages := []Message{
		{Role: "user", Content: "hello"},
	}

	// Case 1: Use model from config
	resp, _ := p.Chat(context.Background(), messages, nil, "gemini-cli", nil)
	if !strings.Contains(resp.Content, "Model used: gemini-2.0-flash") {
		t.Errorf("expected config model, got: %q", resp.Content)
	}

	// Case 2: Override model via argument
	resp, _ = p.Chat(context.Background(), messages, nil, "gemini-2.0-pro", nil)
	if !strings.Contains(resp.Content, "Model used: gemini-2.0-pro") {
		t.Errorf("expected override model, got: %q", resp.Content)
	}
}

func TestGeminiCLIProvider_GetDefaultModel(t *testing.T) {
	p := NewGeminiCLIProvider(config.GeminiCLIConfig{}, nil)
	if p.GetDefaultModel() != "gemini-cli" {
		t.Errorf("expected gemini-cli, got %q", p.GetDefaultModel())
	}
}

func TestGeminiCLIProvider_Chat_Vision(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "mock-gemini-vision-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Script that echoes all arguments
	content := `#!/bin/sh
echo "$*"
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0755)

	cfg := config.GeminiCLIConfig{
		Enabled:    true,
		BinaryPath: tmpFile.Name(),
	}
	p := NewGeminiCLIProvider(cfg, nil)

	messages := []Message{
		{Role: "user", Content: "What is in this image? [image: /tmp/test.jpg]"},
	}

	resp, err := p.Chat(context.Background(), messages, nil, "gemini-cli", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the prompt contains the vision context
	if !strings.Contains(resp.Content, "## Vision Context") {
		t.Errorf("expected vision context in prompt, got: %q", resp.Content)
	}

	// Verify the image path IS in the prompt content
	if !strings.Contains(resp.Content, "/tmp/test.jpg") {
		t.Errorf("expected image path in prompt content, got: %q", resp.Content)
	}
}
func TestGeminiCLIProvider_Chat_RetryAndFallback(t *testing.T) {
	// Create a script that fails with RESOURCEEXHAUSTED until we switch to fallback
	tmpFile, err := os.CreateTemp("", "mock-gemini-retry-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `#!/bin/sh
model=""
while [ $# -gt 0 ]; do
  if [ "$1" = "-m" ]; then
    model="$2"
  fi
  shift
done

if [ "$model" = "gemini-3-pro-preview" ] || [ "$model" = "gemini-3-flash-preview" ]; then
  # Use a string that contains QuotaError (to trigger retry) 
  echo "CRITICAL_ERROR: RetryableQuotaError - MODELCAPACITYEXHAUSTED_LIMIT_HIT"
  exit 1
fi

echo "Success with $model"
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0755)

	msgBus := bus.NewMessageBus()
	cfg := config.GeminiCLIConfig{
		Enabled:    true,
		BinaryPath: tmpFile.Name(),
		Model:      "gemini-3-pro-preview",
	}
	p := NewGeminiCLIProvider(cfg, msgBus)

	messages := []Message{
		{Role: "user", Content: "hello"},
	}

	// Capture outbound messages to verify retry/fallback notifications
	captured := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for {
			msg, ok := msgBus.SubscribeOutbound(ctx)
			if !ok {
				return
			}
			captured <- msg.Content
		}
	}()

	// Use a shorter backoff for testing
	resp, err := p.Chat(ctx, messages, nil, "gemini-cli", map[string]interface{}{
		"channel":      "test",
		"chat_id":      "123",
		"base_backoff": 10 * time.Millisecond,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(resp.Content, "Success with gemini-2.0-flash") {
		t.Errorf("expected success with 2.0-flash fallback model, got: %q", resp.Content)
	}

	// Verify we got notifications
	foundRetry := false
	foundFallback := false
	for i := 0; i < 4; i++ {
		select {
		case msg := <-captured:
			if strings.Contains(msg, "Quota exceeded") {
				foundRetry = true
			}
			if strings.Contains(msg, "Switching to") {
				foundFallback = true
			}
		case <-time.After(30 * time.Second): // Long timeout for exponential backoff
			t.Logf("Timed out waiting for notification %d", i)
			break // Don't fatal, see if we got at least some
		}
	}

	if !foundRetry || !foundFallback {
		t.Errorf("missing expected notifications: foundRetry=%v, foundFallback=%v", foundRetry, foundFallback)
	}
}

func TestFilterOutput(t *testing.T) {
	input := `YOLO mode is enabled. All tool calls will be automatically approved.
Loaded cached credentials.
Hook registry initialized with 0 hook entries
Attempt 1 failed: You have exhausted your capacity...
I will check the weather.
pgrep: bash: line 1: pgrep: command not found
{
  "error": "true",
  "message": "Too many requests"
}
Actual Answer
    at Gaxios.request (/usr/local/lib/node_modules/...)`

	expected := `I will check the weather.
Actual Answer`

	result := filterOutput(input)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}
