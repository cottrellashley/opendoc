package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ToolDefs returns the tool definitions for the LLM.
var ToolDefs = []ToolDefinition{
	{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "read_file",
			Description: "Read the contents of a file in the workspace",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative path from workspace root (e.g. 'content/about.md')",
					},
				},
				"required": []string{"path"},
			},
		},
	},
	{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "write_file",
			Description: "Create or overwrite a file in the workspace. Use this to create new pages, posts, or modify existing content.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative path from workspace root",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Full file content to write",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	},
	{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "edit_file",
			Description: "Make a surgical edit to a file by replacing a specific string. Use for small changes.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative path from workspace root",
					},
					"search": map[string]any{
						"type":        "string",
						"description": "Exact string to find in the file",
					},
					"replace": map[string]any{
						"type":        "string",
						"description": "String to replace it with",
					},
				},
				"required": []string{"path", "search", "replace"},
			},
		},
	},
	{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "list_files",
			Description: "List files and directories in a given path",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative directory path (e.g. 'content/' or '.')",
					},
				},
				"required": []string{"path"},
			},
		},
	},
	{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "build",
			Description: "Trigger a build to regenerate the site from the current content. Call this after making changes.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	},
	{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "get_config",
			Description: "Read and return the current opendoc.yml configuration",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	},
	{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "update_nav",
			Description: "Update the navigation items in opendoc.yml. Pass the full nav array.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"items": map[string]any{
						"type":        "array",
						"description": "Navigation items, each an object with a single key (label) and value (path). Example: [{'Home': 'index.md'}, {'About': 'about.md'}]",
						"items":       map[string]any{"type": "object"},
					},
				},
				"required": []string{"items"},
			},
		},
	},
}

// BuildFunc is the type of the build trigger function.
type BuildFunc func() map[string]any

// ExecuteTool runs a tool and returns the JSON result string.
func ExecuteTool(toolName string, args map[string]any, workspace string, buildFn BuildFunc) string {
	switch toolName {
	case "read_file":
		return toolReadFile(getString(args, "path"), workspace)
	case "write_file":
		return toolWriteFile(getString(args, "path"), getString(args, "content"), workspace)
	case "edit_file":
		return toolEditFile(getString(args, "path"), getString(args, "search"), getString(args, "replace"), workspace)
	case "list_files":
		return toolListFiles(getString(args, "path"), workspace)
	case "build":
		return toolBuild(buildFn)
	case "get_config":
		return toolGetConfig(workspace)
	case "update_nav":
		items, _ := args["items"].([]any)
		return toolUpdateNav(items, workspace)
	default:
		return jsonStr(map[string]string{"error": "Unknown tool: " + toolName})
	}
}

// ── Individual tool implementations ─────────────────────────

func toolReadFile(relPath, workspace string) string {
	absPath := filepath.Join(workspace, relPath)
	if !strings.HasPrefix(absPath, workspace) {
		return jsonStr(map[string]string{"error": "Access denied"})
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return jsonStr(map[string]string{"error": "File not found: " + relPath})
	}
	return jsonStr(map[string]any{"path": relPath, "content": string(data)})
}

func toolWriteFile(relPath, content, workspace string) string {
	absPath := filepath.Join(workspace, relPath)
	if !strings.HasPrefix(absPath, workspace) {
		return jsonStr(map[string]string{"error": "Access denied"})
	}
	dir := filepath.Dir(absPath)
	os.MkdirAll(dir, 0o755)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return jsonStr(map[string]string{"error": err.Error()})
	}
	return jsonStr(map[string]any{"success": true, "path": relPath, "size": len(content)})
}

func toolEditFile(relPath, search, replace, workspace string) string {
	absPath := filepath.Join(workspace, relPath)
	if !strings.HasPrefix(absPath, workspace) {
		return jsonStr(map[string]string{"error": "Access denied"})
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return jsonStr(map[string]string{"error": "File not found: " + relPath})
	}
	content := string(data)
	if !strings.Contains(content, search) {
		return jsonStr(map[string]string{"error": "Search string not found in " + relPath})
	}
	newContent := strings.Replace(content, search, replace, 1)
	os.WriteFile(absPath, []byte(newContent), 0o644)
	return jsonStr(map[string]any{"success": true, "path": relPath})
}

func toolListFiles(relPath, workspace string) string {
	if relPath == "" {
		relPath = "."
	}
	absPath := filepath.Join(workspace, relPath)
	if !strings.HasPrefix(absPath, workspace) {
		return jsonStr(map[string]string{"error": "Access denied"})
	}
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return jsonStr(map[string]string{"error": "Directory not found: " + relPath})
	}

	var items []map[string]string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "dist" {
			continue
		}
		typ := "file"
		if e.IsDir() {
			typ = "directory"
		}
		items = append(items, map[string]string{"name": name, "type": typ})
	}
	return jsonStr(map[string]any{"path": relPath, "entries": items})
}

func toolBuild(buildFn BuildFunc) string {
	result := buildFn()
	return jsonStr(result)
}

func toolGetConfig(workspace string) string {
	configPath := filepath.Join(workspace, "opendoc.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return jsonStr(map[string]string{"error": "No opendoc.yml found"})
	}
	return jsonStr(map[string]any{"config": string(data)})
}

func toolUpdateNav(items []any, workspace string) string {
	configPath := filepath.Join(workspace, "opendoc.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return jsonStr(map[string]string{"error": "No opendoc.yml found"})
	}
	var raw map[string]any
	yaml.Unmarshal(data, &raw)
	raw["nav"] = items
	newData, _ := yaml.Marshal(raw)
	os.WriteFile(configPath, newData, 0o644)
	return jsonStr(map[string]any{"success": true, "nav": items})
}

// ── Helpers ─────────────────────────────────────────────────

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func jsonStr(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}
