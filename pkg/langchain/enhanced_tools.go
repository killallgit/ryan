package langchain

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// EnhancedToolAdapter wraps Ryan tools with file context awareness
type EnhancedToolAdapter struct {
	baseTool      tools.Tool
	fileMemory    *FileContextMemory
	vectorManager *VectorStoreManager
	log           *logger.Logger
}

// Name implements the LangChain Tool interface
func (eta *EnhancedToolAdapter) Name() string {
	return eta.baseTool.Name()
}

// Description implements the LangChain Tool interface
func (eta *EnhancedToolAdapter) Description() string {
	return eta.baseTool.Description() + " (Enhanced with file context awareness)"
}

// Call implements the LangChain Tool interface
func (eta *EnhancedToolAdapter) Call(ctx context.Context, input string) (string, error) {
	eta.log.Debug("Enhanced tool call", "tool", eta.baseTool.Name(), "input_length", len(input))

	// Parse input with better JSON handling
	params, err := eta.parseEnhancedInput(input)
	if err != nil {
		eta.log.Error("Failed to parse tool input", "error", err, "input", input)
		return "", fmt.Errorf("parsing input: %w", err)
	}

	// Add file context if relevant
	if eta.isFileRelevantTool() {
		params = eta.addFileContext(params)
	}

	// Execute the base tool
	result, err := eta.baseTool.Execute(ctx, params)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("tool execution failed: %s", result.Error)
	}

	// Post-process result if it's a file operation
	eta.postProcessResult(params, &result)

	return result.Content, nil
}

// parseEnhancedInput parses tool input with better JSON handling
func (eta *EnhancedToolAdapter) parseEnhancedInput(input string) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Try to parse as JSON first
	if err := json.Unmarshal([]byte(input), &params); err == nil {
		return params, nil
	}

	// Fall back to key-value parsing
	switch eta.baseTool.Name() {
	case "execute_bash":
		params["command"] = input
	case "read_file":
		params["path"] = input
	case "write_file":
		// Try to extract path and content
		if parsed := eta.parseWriteFileInput(input); parsed != nil {
			return parsed, nil
		}
		params["path"] = input
	case "edit_file":
		if parsed := eta.parseEditFileInput(input); parsed != nil {
			return parsed, nil
		}
		params["path"] = input
	default:
		params["input"] = input
	}

	return params, nil
}

// parseWriteFileInput attempts to parse write file input
func (eta *EnhancedToolAdapter) parseWriteFileInput(input string) map[string]interface{} {
	// Look for patterns like "path: file.txt\ncontent: content here"
	lines := splitLines(input)
	params := make(map[string]interface{})

	for _, line := range lines {
		if colon := findFirst(line, ":"); colon != -1 {
			key := trimSpace(line[:colon])
			value := trimSpace(line[colon+1:])
			params[key] = value
		}
	}

	if _, hasPath := params["path"]; hasPath {
		return params
	}

	return nil
}

// parseEditFileInput attempts to parse edit file input
func (eta *EnhancedToolAdapter) parseEditFileInput(input string) map[string]interface{} {
	params := make(map[string]interface{})
	lines := splitLines(input)

	for _, line := range lines {
		if colon := findFirst(line, ":"); colon != -1 {
			key := trimSpace(line[:colon])
			value := trimSpace(line[colon+1:])

			// Convert numeric values
			if key == "start_line" || key == "end_line" {
				if num, err := strconv.Atoi(value); err == nil {
					params[key] = num
				}
			} else {
				params[key] = value
			}
		}
	}

	if _, hasPath := params["path"]; hasPath {
		return params
	}

	return nil
}

// addFileContext adds relevant file context to parameters
func (eta *EnhancedToolAdapter) addFileContext(params map[string]interface{}) map[string]interface{} {
	if path, ok := params["path"].(string); ok {
		if fc, exists := eta.fileMemory.GetFileContext(path); exists {
			params["file_context"] = fc
		}
	}

	// Add all file contexts for broader context
	if eta.fileMemory != nil {
		contexts := eta.fileMemory.GetRelevantFileContexts("")
		if len(contexts) > 0 {
			params["available_files"] = contexts
		}
	}

	return params
}

// isFileRelevantTool checks if the tool is file-related
func (eta *EnhancedToolAdapter) isFileRelevantTool() bool {
	name := eta.baseTool.Name()
	fileTools := []string{"read_file", "write_file", "edit_file", "execute_bash"}

	for _, tool := range fileTools {
		if name == tool {
			return true
		}
	}

	return false
}

// postProcessResult handles post-processing of tool results
func (eta *EnhancedToolAdapter) postProcessResult(params map[string]interface{}, result *tools.ToolResult) {
	if !eta.isFileRelevantTool() {
		return
	}

	// If this was a file operation, update file context
	if path, ok := params["path"].(string); ok {
		switch eta.baseTool.Name() {
		case "write_file", "edit_file":
			// File was modified, update context
			eta.updateFileContextFromResult(path, params, result)
		}
	}
}

// updateFileContextFromResult updates file context after a file operation
func (eta *EnhancedToolAdapter) updateFileContextFromResult(path string, params map[string]interface{}, result *tools.ToolResult) {
	fc, exists := eta.fileMemory.GetFileContext(path)
	if !exists {
		fc = &chat.FileContext{
			Path:        path,
			EditHistory: []chat.FileEdit{},
		}
	}

	// Create edit record
	edit := chat.FileEdit{
		Timestamp: timeNow(),
		EditType:  eta.baseTool.Name(),
	}

	if content, ok := params["content"].(string); ok {
		edit.NewContent = content
	}

	fc.EditHistory = append(fc.EditHistory, edit)
	fc.LastEdit = edit.Timestamp

	eta.fileMemory.AddFileContext(*fc)
}

// FileProcessorTool wraps the file processing chain as a LangChain tool
type FileProcessorTool struct {
	processor *FileProcessingChain
	log       *logger.Logger
}

func (fpt *FileProcessorTool) Name() string {
	return "process_file"
}

func (fpt *FileProcessorTool) Description() string {
	return "Process a file for embedding and context tracking. Input: {\"file_path\": \"path/to/file\"}"
}

func (fpt *FileProcessorTool) Call(ctx context.Context, input string) (string, error) {
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		params = map[string]interface{}{"file_path": input}
	}

	result, err := fpt.processor.Call(ctx, params)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("File processed successfully. Chunks: %v, Hash: %v",
		result["chunks_count"], result["content_hash"]), nil
}

// FileEditorTool wraps the file editor chain as a LangChain tool
type FileEditorTool struct {
	editor *FileEditChain
	log    *logger.Logger
}

func (fet *FileEditorTool) Name() string {
	return "edit_file_advanced"
}

func (fet *FileEditorTool) Description() string {
	return "Advanced file editing with history tracking. Input: {\"operation\": {\"type\": \"replace|insert|delete\", \"file_path\": \"path\", \"old_content\": \"old\", \"new_content\": \"new\"}}"
}

func (fet *FileEditorTool) Call(ctx context.Context, input string) (string, error) {
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	result, err := fet.editor.Call(ctx, params)
	if err != nil {
		return "", err
	}

	if editResult, ok := result["result"].(*EditResult); ok {
		if editResult.Success {
			return fmt.Sprintf("File edited successfully. Type: %s, Lines changed: %d",
				editResult.EditType, len(editResult.LinesChanged)), nil
		} else {
			return fmt.Sprintf("Edit failed: %s", editResult.Error), nil
		}
	}

	return "Edit completed", nil
}

// FileSearchTool wraps the file search chain as a LangChain tool
type FileSearchTool struct {
	searcher *FileSearchChain
	log      *logger.Logger
}

func (fst *FileSearchTool) Name() string {
	return "search_files"
}

func (fst *FileSearchTool) Description() string {
	return "Search for content across all files using semantic search. Input: {\"query\": \"search terms\", \"num_results\": 5}"
}

func (fst *FileSearchTool) Call(ctx context.Context, input string) (string, error) {
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		params = map[string]interface{}{"query": input, "num_results": 5}
	}

	result, err := fst.searcher.Call(ctx, params)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Found %v results using %v search",
		result["num_found"], result["search_type"]), nil
}

// ConversationBranchTool provides branching operations
type ConversationBranchTool struct {
	memory *BranchingConversationMemory
	log    *logger.Logger
}

func (cbt *ConversationBranchTool) Name() string {
	return "branch_conversation"
}

func (cbt *ConversationBranchTool) Description() string {
	return "Create or switch conversation branches. Input: {\"action\": \"create|switch|list\", \"message_id\": \"id\", \"branch_name\": \"name\"}"
}

func (cbt *ConversationBranchTool) Call(ctx context.Context, input string) (string, error) {
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	action, ok := params["action"].(string)
	if !ok {
		return "", fmt.Errorf("action parameter required")
	}

	switch action {
	case "create":
		messageID, _ := params["message_id"].(string)
		branchName, _ := params["branch_name"].(string)

		_, err := cbt.memory.branchingConv.BranchFrom(messageID, branchName)
		if err != nil {
			return "", fmt.Errorf("creating branch: %w", err)
		}

		return fmt.Sprintf("Created branch: %s", branchName), nil

	case "switch":
		branchID, ok := params["branch_id"].(string)
		if !ok {
			return "", fmt.Errorf("branch_id parameter required for switch")
		}

		_, err := cbt.memory.branchingConv.SwitchBranch(branchID)
		if err != nil {
			return "", fmt.Errorf("switching branch: %w", err)
		}

		return fmt.Sprintf("Switched to branch: %s", branchID), nil

	case "list":
		branches := cbt.memory.branchingConv.GetAllBranches()
		var branchNames []string
		for _, branch := range branches {
			branchNames = append(branchNames, fmt.Sprintf("%s (%s)", branch.Name, branch.ID))
		}

		return fmt.Sprintf("Available branches: %v", branchNames), nil

	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

// Helper functions for string manipulation
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	
	lines := make([]string, 0)
	start := 0
	
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	
	return lines
}

func findFirst(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	
	return s[start:end]
}

func timeNow() time.Time {
	return time.Now()
}