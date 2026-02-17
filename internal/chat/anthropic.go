package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/cottrellashley/opendoc/internal/core"
)

// AnthropicAdapter implements LLMAdapter for Claude via direct HTTP.
type AnthropicAdapter struct {
	apiKey string
	model  string
}

// NewAnthropicAdapter creates a new Anthropic adapter.
func NewAnthropicAdapter() *AnthropicAdapter {
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &AnthropicAdapter{
		apiKey: core.ResolveAPIKey("anthropic"),
		model:  model,
	}
}

func (a *AnthropicAdapter) Name() string { return "anthropic" }

func (a *AnthropicAdapter) Chat(messages []ChatMessage, tools []ToolDefinition, onChunk func(StreamChunk)) (ChatMessage, error) {
	// Extract system message
	var systemContent string
	var nonSystemMsgs []ChatMessage
	for _, msg := range messages {
		if msg.Role == "system" {
			systemContent = msg.Content
		} else {
			nonSystemMsgs = append(nonSystemMsgs, msg)
		}
	}

	// Build Anthropic-format messages
	var apiMsgs []map[string]any
	for _, msg := range nonSystemMsgs {
		switch msg.Role {
		case "tool":
			apiMsgs = append(apiMsgs, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": msg.ToolCallID,
					"content":     msg.Content,
				}},
			})
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				var blocks []map[string]any
				if msg.Content != "" {
					blocks = append(blocks, map[string]any{"type": "text", "text": msg.Content})
				}
				for _, tc := range msg.ToolCalls {
					var input map[string]any
					json.Unmarshal([]byte(tc.Function.Arguments), &input)
					if input == nil {
						input = map[string]any{}
					}
					blocks = append(blocks, map[string]any{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Function.Name,
						"input": input,
					})
				}
				apiMsgs = append(apiMsgs, map[string]any{"role": "assistant", "content": blocks})
			} else {
				apiMsgs = append(apiMsgs, map[string]any{"role": "assistant", "content": msg.Content})
			}
		default:
			apiMsgs = append(apiMsgs, map[string]any{"role": msg.Role, "content": msg.Content})
		}
	}

	// Build tools
	var apiTools []map[string]any
	for _, t := range tools {
		apiTools = append(apiTools, map[string]any{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": t.Function.Parameters,
		})
	}

	// Build request body
	body := map[string]any{
		"model":      a.model,
		"max_tokens": 4096,
		"messages":   apiMsgs,
		"stream":     true,
	}
	if systemContent != "" {
		body["system"] = systemContent
	}
	if len(apiTools) > 0 {
		body["tools"] = apiTools
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return ChatMessage{}, err
	}

	// Make streaming request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyJSON))
	if err != nil {
		return ChatMessage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ChatMessage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return ChatMessage{}, fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	var fullText string
	var toolCalls []ToolCall
	currentToolID := ""
	currentToolName := ""
	currentToolArgs := ""

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "content_block_start":
			if cb, ok := event["content_block"].(map[string]any); ok {
				if cbType, _ := cb["type"].(string); cbType == "tool_use" {
					currentToolID, _ = cb["id"].(string)
					currentToolName, _ = cb["name"].(string)
					currentToolArgs = ""
				}
			}

		case "content_block_delta":
			if delta, ok := event["delta"].(map[string]any); ok {
				deltaType, _ := delta["type"].(string)
				switch deltaType {
				case "text_delta":
					text, _ := delta["text"].(string)
					fullText += text
					onChunk(StreamChunk{Type: "text", Content: text})
				case "input_json_delta":
					partial, _ := delta["partial_json"].(string)
					currentToolArgs += partial
				}
			}

		case "content_block_stop":
			if currentToolID != "" {
				tc := ToolCall{
					ID:   currentToolID,
					Type: "function",
					Function: FunctionCall{
						Name:      currentToolName,
						Arguments: currentToolArgs,
					},
				}
				toolCalls = append(toolCalls, tc)
				onChunk(StreamChunk{Type: "tool_call_start", ToolCall: &tc})
				onChunk(StreamChunk{Type: "tool_call_end", ToolCall: &tc})
				currentToolID = ""
				currentToolName = ""
				currentToolArgs = ""
			}
		}
	}

	onChunk(StreamChunk{Type: "done"})

	result := ChatMessage{
		Role:    "assistant",
		Content: fullText,
	}
	if len(toolCalls) > 0 {
		result.ToolCalls = toolCalls
	}
	return result, nil
}
