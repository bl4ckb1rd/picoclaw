package providers

import (
	"context"
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestGeminiCLIProvider_Chat_NoUserMessage(t *testing.T) {
	cfg := config.GeminiCLIConfig{
		Enabled:    true,
		BinaryPath: "echo",
	}
	p := NewGeminiCLIProvider(cfg)

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
echo "Mock Gemini Output for prompt: $2"
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
	p := NewGeminiCLIProvider(cfg)

	messages := []Message{
		{Role: "user", Content: "hello gemini"},
	}

	resp, err := p.Chat(context.Background(), messages, nil, "gemini-cli", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `Mock Gemini Output for prompt: hello gemini
`
	if resp.Content != expected {
		t.Errorf("expected %q, got %q", expected, resp.Content)
	}
}
