package providers

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

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

func TestFilterOutput(t *testing.T) {
	input := `YOLO mode is enabled. All tool calls will be automatically approved.
Loaded cached credentials.
Hook registry initialized with 0 hook entries
Attempt 1 failed: You have exhausted your capacity...
I will check the weather.
pgrep: bash: line 1: pgrep: command not found
Actual Answer`

	expected := `I will check the weather.
Actual Answer`

	result := filterOutput(input)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}
