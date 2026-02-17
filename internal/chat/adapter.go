// Package chat implements the LLM chat system for the OpenDoc workbench.
package chat

// ChatMessage represents a message in the conversation.
type ChatMessage struct {
	Role       string     `json:"role"` // "system", "user", "assistant", "tool"
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a function call requested by the LLM.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall holds the function name and JSON arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolDefinition defines a tool the LLM can call.
type ToolDefinition struct {
	Type     string             `json:"type"` // "function"
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition holds the function schema.
type FunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// StreamChunk represents a piece of streamed response.
type StreamChunk struct {
	Type     string    `json:"type"` // "text", "tool_call_start", "tool_call_args", "tool_call_end", "done"
	Content  string    `json:"content,omitempty"`
	ToolCall *ToolCall `json:"toolCall,omitempty"`
}

// LLMAdapter defines the interface for LLM providers.
type LLMAdapter interface {
	// Name returns the adapter identifier.
	Name() string

	// Chat sends messages with tool definitions and streams back the response.
	Chat(messages []ChatMessage, tools []ToolDefinition, onChunk func(StreamChunk)) (ChatMessage, error)
}
