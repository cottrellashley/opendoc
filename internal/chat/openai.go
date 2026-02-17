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

// OpenAIAdapter implements LLMAdapter for GPT models via direct HTTP.
type OpenAIAdapter struct {
	apiKey string
	model  string
}

// NewOpenAIAdapter creates a new OpenAI adapter.
func NewOpenAIAdapter() *OpenAIAdapter {
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIAdapter{
		apiKey: core.ResolveAPIKey("openai"),
		model:  model,
	}
}

func (a *OpenAIAdapter) Name() string { return "openai" }

func (a *OpenAIAdapter) Chat(messages []ChatMessage, tools []ToolDefinition, onChunk func(StreamChunk)) (ChatMessage, error) {
	// Build OpenAI-format messages
	var apiMsgs []map[string]any
	for _, msg := range messages {
		m := map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.Role == "tool" {
			m["tool_call_id"] = msg.ToolCallID
		}
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			var tcs []map[string]any
			for _, tc := range msg.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			m["tool_calls"] = tcs
		}
		apiMsgs = append(apiMsgs, m)
	}

	// Build tools
	var apiTools []map[string]any
	for _, t := range tools {
		apiTools = append(apiTools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Function.Name,
				"description": t.Function.Description,
				"parameters":  t.Function.Parameters,
			},
		})
	}

	// Build request body
	body := map[string]any{
		"model":    a.model,
		"messages": apiMsgs,
		"stream":   true,
	}
	if len(apiTools) > 0 {
		body["tools"] = apiTools
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return ChatMessage{}, err
	}

	// Make streaming request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyJSON))
	if err != nil {
		return ChatMessage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ChatMessage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return ChatMessage{}, fmt.Errorf("openai API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	var fullText string
	toolCallAccumulators := make(map[int]*struct {
		id   string
		name string
		args string
	})

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

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta

		if delta.Content != "" {
			fullText += delta.Content
			onChunk(StreamChunk{Type: "text", Content: delta.Content})
		}

		for _, tc := range delta.ToolCalls {
			idx := tc.Index
			if _, ok := toolCallAccumulators[idx]; !ok {
				toolCallAccumulators[idx] = &struct {
					id   string
					name string
					args string
				}{
					id:   tc.ID,
					name: tc.Function.Name,
				}
				onChunk(StreamChunk{
					Type: "tool_call_start",
					ToolCall: &ToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: FunctionCall{Name: tc.Function.Name},
					},
				})
			}
			if tc.Function.Arguments != "" {
				toolCallAccumulators[idx].args += tc.Function.Arguments
				onChunk(StreamChunk{Type: "tool_call_args", Content: tc.Function.Arguments})
			}
		}
	}

	// Finalize tool calls
	var toolCalls []ToolCall
	for _, acc := range toolCallAccumulators {
		tc := ToolCall{
			ID:   acc.id,
			Type: "function",
			Function: FunctionCall{
				Name:      acc.name,
				Arguments: acc.args,
			},
		}
		toolCalls = append(toolCalls, tc)
		onChunk(StreamChunk{Type: "tool_call_end", ToolCall: &tc})
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
