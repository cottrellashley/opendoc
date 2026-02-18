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
	"sync"
	"time"
)

const (
	copilotTokenURL    = "https://api.github.com/copilot_internal/v2/token"
	copilotDefaultModel = "gpt-4.1"
	copilotUserAgent   = "GithubCopilot/1.155.0"
)

// CopilotAdapter implements LLMAdapter for GitHub Copilot.
type CopilotAdapter struct {
	oauthToken  string
	model       string
	mu          sync.Mutex
	apiToken    string
	apiEndpoint string
	tokenExpiry time.Time
}

type copilotTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	RefreshIn int64  `json:"refresh_in"`
	Endpoints struct {
		API string `json:"api"`
	} `json:"endpoints"`
}

// NewCopilotAdapter creates a new Copilot adapter with the given OAuth token.
// If model is empty, falls back to COPILOT_MODEL env var or the default.
func NewCopilotAdapter(oauthToken, model string) *CopilotAdapter {
	if model == "" {
		model = os.Getenv("COPILOT_MODEL")
	}
	if model == "" {
		model = copilotDefaultModel
	}
	return &CopilotAdapter{
		oauthToken: oauthToken,
		model:      model,
	}
}

func (a *CopilotAdapter) Name() string { return "copilot" }

// getAPIToken returns a valid short-lived Copilot API token, refreshing if needed.
func (a *CopilotAdapter) getAPIToken() (endpoint, token string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.apiToken != "" && time.Now().Before(a.tokenExpiry) {
		return a.apiEndpoint, a.apiToken, nil
	}

	req, err := http.NewRequest("GET", copilotTokenURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Authorization", "bearer "+a.oauthToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", copilotUserAgent)
	req.Header.Set("editor-version", "vscode/1.85.1")
	req.Header.Set("editor-plugin-version", "copilot/1.155.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("copilot token error (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp copilotTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", fmt.Errorf("parse token response: %w", err)
	}

	a.apiToken = tokenResp.Token
	a.apiEndpoint = tokenResp.Endpoints.API
	if tokenResp.RefreshIn > 0 {
		a.tokenExpiry = time.Now().Add(time.Duration(tokenResp.RefreshIn) * time.Second)
	} else {
		a.tokenExpiry = time.Now().Add(25 * time.Minute)
	}

	return a.apiEndpoint, a.apiToken, nil
}

func (a *CopilotAdapter) Chat(messages []ChatMessage, tools []ToolDefinition, onChunk func(StreamChunk)) (ChatMessage, error) {
	endpoint, token, err := a.getAPIToken()
	if err != nil {
		return ChatMessage{}, fmt.Errorf("copilot auth: %w", err)
	}

	// Build OpenAI-format messages (same format as OpenAI adapter)
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

	chatURL := strings.TrimRight(endpoint, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", chatURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return ChatMessage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", copilotUserAgent)
	req.Header.Set("editor-version", "vscode/1.85.1")
	req.Header.Set("editor-plugin-version", "copilot/1.155.0")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ChatMessage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return ChatMessage{}, fmt.Errorf("copilot API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse SSE stream — identical to OpenAI format
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
