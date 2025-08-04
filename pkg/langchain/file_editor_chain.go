package langchain

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/schema"
)

// FileEditChain handles file editing with history tracking and diff generation
type FileEditChain struct {
	memory *FileContextMemory
	log    *logger.Logger
}

// EditOperation represents a file edit operation
type EditOperation struct {
	Type        string `json:"type"`         // "replace", "insert", "delete", "create"
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line,omitempty"`
	EndLine     int    `json:"end_line,omitempty"`
	OldContent  string `json:"old_content,omitempty"`
	NewContent  string `json:"new_content"`
	Description string `json:"description,omitempty"`
}

// EditResult represents the result of an edit operation
type EditResult struct {
	Success      bool              `json:"success"`
	FilePath     string            `json:"file_path"`
	EditType     string            `json:"edit_type"`
	LinesChanged []chat.LineChange `json:"lines_changed"`
	DiffPatch    string            `json:"diff_patch"`
	Error        string            `json:"error,omitempty"`
	FileContext  *chat.FileContext `json:"file_context,omitempty"`
}

// NewFileEditChain creates a new file editor chain
func NewFileEditChain(memory *FileContextMemory) *FileEditChain {
	return &FileEditChain{
		memory: memory,
		log:    logger.WithComponent("file_edit_chain"),
	}
}

// Call implements the Chain interface
func (c *FileEditChain) Call(ctx context.Context, inputs map[string]any, options ...chains.ChainCallOption) (map[string]any, error) {
	operation, ok := inputs["operation"].(EditOperation)
	if !ok {
		return nil, fmt.Errorf("operation input required")
	}

	messageID := ""
	if mid, ok := inputs["message_id"].(string); ok {
		messageID = mid
	}

	// Execute the edit operation
	result, err := c.ExecuteEdit(ctx, operation, messageID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"result":       result,
		"success":      result.Success,
		"file_context": result.FileContext,
	}, nil
}

// ExecuteEdit performs a file edit operation with history tracking
func (c *FileEditChain) ExecuteEdit(ctx context.Context, op EditOperation, messageID string) (*EditResult, error) {
	c.log.Debug("Executing edit operation", "type", op.Type, "file", op.FilePath)

	result := &EditResult{
		FilePath: op.FilePath,
		EditType: op.Type,
	}

	switch op.Type {
	case "create":
		return c.createFile(op, messageID)
	case "replace":
		return c.replaceContent(op, messageID)
	case "insert":
		return c.insertContent(op, messageID)
	case "delete":
		return c.deleteContent(op, messageID)
	default:
		result.Error = fmt.Sprintf("unsupported edit type: %s", op.Type)
		return result, nil
	}
}

// createFile creates a new file
func (c *FileEditChain) createFile(op EditOperation, messageID string) (*EditResult, error) {
	result := &EditResult{
		FilePath: op.FilePath,
		EditType: "create",
	}

	// Check if file already exists
	if _, err := os.Stat(op.FilePath); err == nil {
		result.Error = "file already exists"
		return result, nil
	}

	// Create file
	if err := os.WriteFile(op.FilePath, []byte(op.NewContent), 0644); err != nil {
		result.Error = fmt.Sprintf("failed to create file: %v", err)
		return result, nil
	}

	// Create file context
	fc := chat.FileContext{
		Path:        op.FilePath,
		Content:     op.NewContent,
		ContentHash: c.hashContent([]byte(op.NewContent)),
		LastEdit:    time.Now(),
		EditHistory: []chat.FileEdit{
			{
				MessageID: chat.MessageID{
					ID:        messageID,
					Timestamp: time.Now(),
				},
				Timestamp:  time.Now(),
				NewContent: op.NewContent,
				EditType:   "create",
				DiffPatch:  c.generateCreateDiff(op.NewContent),
			},
		},
	}

	// Add to memory
	c.memory.AddFileContext(fc)

	result.Success = true
	result.FileContext = &fc
	result.DiffPatch = fc.EditHistory[0].DiffPatch

	return result, nil
}

// replaceContent replaces content in a file
func (c *FileEditChain) replaceContent(op EditOperation, messageID string) (*EditResult, error) {
	result := &EditResult{
		FilePath: op.FilePath,
		EditType: "replace",
	}

	// Read current content
	currentContent, err := os.ReadFile(op.FilePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		return result, nil
	}

	oldContentStr := string(currentContent)
	
	// Perform replacement
	var newContent string
	if op.OldContent != "" {
		// Replace specific content
		newContent = strings.ReplaceAll(oldContentStr, op.OldContent, op.NewContent)
		if newContent == oldContentStr {
			result.Error = "old content not found in file"
			return result, nil
		}
	} else {
		// Replace entire file
		newContent = op.NewContent
	}

	// Write new content
	if err := os.WriteFile(op.FilePath, []byte(newContent), 0644); err != nil {
		result.Error = fmt.Sprintf("failed to write file: %v", err)
		return result, nil
	}

	// Generate diff and line changes
	lineChanges := c.generateLineChanges(oldContentStr, newContent)
	diffPatch := c.generateDiff(oldContentStr, newContent)

	// Update file context
	fc, exists := c.memory.GetFileContext(op.FilePath)
	if !exists {
		// Create new context
		fc = &chat.FileContext{
			Path:        op.FilePath,
			EditHistory: []chat.FileEdit{},
		}
	}

	// Add edit to history
	edit := chat.FileEdit{
		MessageID: chat.MessageID{
			ID:        messageID,
			Timestamp: time.Now(),
		},
		Timestamp:   time.Now(),
		OldContent:  oldContentStr,
		NewContent:  newContent,
		EditType:    "replace",
		DiffPatch:   diffPatch,
		LineChanges: lineChanges,
	}

	fc.Content = newContent
	fc.ContentHash = c.hashContent([]byte(newContent))
	fc.LastEdit = time.Now()
	fc.EditHistory = append(fc.EditHistory, edit)

	// Update memory
	c.memory.AddFileContext(*fc)

	result.Success = true
	result.FileContext = fc
	result.DiffPatch = diffPatch
	result.LinesChanged = lineChanges

	return result, nil
}

// insertContent inserts content at a specific line
func (c *FileEditChain) insertContent(op EditOperation, messageID string) (*EditResult, error) {
	result := &EditResult{
		FilePath: op.FilePath,
		EditType: "insert",
	}

	// Read current content
	currentContent, err := os.ReadFile(op.FilePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		return result, nil
	}

	lines := strings.Split(string(currentContent), "\n")
	
	// Insert content at specified line
	insertLine := op.StartLine - 1 // Convert to 0-based index
	if insertLine < 0 || insertLine > len(lines) {
		result.Error = fmt.Sprintf("invalid line number: %d", op.StartLine)
		return result, nil
	}

	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertLine]...)
	
	// Insert new content (split by lines if multi-line)
	insertLines := strings.Split(op.NewContent, "\n")
	newLines = append(newLines, insertLines...)
	
	newLines = append(newLines, lines[insertLine:]...)
	
	newContent := strings.Join(newLines, "\n")

	// Write new content
	if err := os.WriteFile(op.FilePath, []byte(newContent), 0644); err != nil {
		result.Error = fmt.Sprintf("failed to write file: %v", err)
		return result, nil
	}

	// Update file context (similar to replace)
	return c.updateFileContextAfterEdit(op.FilePath, string(currentContent), newContent, "insert", messageID)
}

// deleteContent deletes content from a file
func (c *FileEditChain) deleteContent(op EditOperation, messageID string) (*EditResult, error) {
	result := &EditResult{
		FilePath: op.FilePath,
		EditType: "delete",
	}

	// Read current content
	currentContent, err := os.ReadFile(op.FilePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		return result, nil
	}

	lines := strings.Split(string(currentContent), "\n")
	
	// Delete lines
	startLine := op.StartLine - 1 // Convert to 0-based
	endLine := op.EndLine - 1
	
	if startLine < 0 || endLine >= len(lines) || startLine > endLine {
		result.Error = fmt.Sprintf("invalid line range: %d-%d", op.StartLine, op.EndLine)
		return result, nil
	}

	newLines := make([]string, 0, len(lines)-(endLine-startLine+1))
	newLines = append(newLines, lines[:startLine]...)
	newLines = append(newLines, lines[endLine+1:]...)
	
	newContent := strings.Join(newLines, "\n")

	// Write new content
	if err := os.WriteFile(op.FilePath, []byte(newContent), 0644); err != nil {
		result.Error = fmt.Sprintf("failed to write file: %v", err)
		return result, nil
	}

	// Update file context
	return c.updateFileContextAfterEdit(op.FilePath, string(currentContent), newContent, "delete", messageID)
}

// Helper methods

func (c *FileEditChain) updateFileContextAfterEdit(filePath, oldContent, newContent, editType, messageID string) (*EditResult, error) {
	lineChanges := c.generateLineChanges(oldContent, newContent)
	diffPatch := c.generateDiff(oldContent, newContent)

	fc, exists := c.memory.GetFileContext(filePath)
	if !exists {
		fc = &chat.FileContext{
			Path:        filePath,
			EditHistory: []chat.FileEdit{},
		}
	}

	edit := chat.FileEdit{
		MessageID: chat.MessageID{
			ID:        messageID,
			Timestamp: time.Now(),
		},
		Timestamp:   time.Now(),
		OldContent:  oldContent,
		NewContent:  newContent,
		EditType:    editType,
		DiffPatch:   diffPatch,
		LineChanges: lineChanges,
	}

	fc.Content = newContent
	fc.ContentHash = c.hashContent([]byte(newContent))
	fc.LastEdit = time.Now()
	fc.EditHistory = append(fc.EditHistory, edit)

	c.memory.AddFileContext(*fc)

	return &EditResult{
		Success:      true,
		FilePath:     filePath,
		EditType:     editType,
		LinesChanged: lineChanges,
		DiffPatch:    diffPatch,
		FileContext:  fc,
	}, nil
}

func (c *FileEditChain) generateLineChanges(oldContent, newContent string) []chat.LineChange {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var changes []chat.LineChange

	// Simple diff algorithm - compare line by line
	minLen := len(oldLines)
	if len(newLines) < minLen {
		minLen = len(newLines)
	}

	// Check for modifications
	for i := 0; i < minLen; i++ {
		if oldLines[i] != newLines[i] {
			changes = append(changes, chat.LineChange{
				LineNumber: i + 1,
				OldLine:    oldLines[i],
				NewLine:    newLines[i],
				ChangeType: "modify",
			})
		}
	}

	// Check for additions
	if len(newLines) > len(oldLines) {
		for i := len(oldLines); i < len(newLines); i++ {
			changes = append(changes, chat.LineChange{
				LineNumber: i + 1,
				NewLine:    newLines[i],
				ChangeType: "add",
			})
		}
	}

	// Check for deletions
	if len(oldLines) > len(newLines) {
		for i := len(newLines); i < len(oldLines); i++ {
			changes = append(changes, chat.LineChange{
				LineNumber: i + 1,
				OldLine:    oldLines[i],
				ChangeType: "delete",
			})
		}
	}

	return changes
}

func (c *FileEditChain) generateDiff(oldContent, newContent string) string {
	// Simple unified diff format
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var diff strings.Builder
	diff.WriteString("--- old\n")
	diff.WriteString("+++ new\n")

	// Generate context for changes
	for i := 0; i < len(oldLines) || i < len(newLines); i++ {
		oldLine := ""
		newLine := ""

		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			if oldLine != "" {
				diff.WriteString(fmt.Sprintf("-%s\n", oldLine))
			}
			if newLine != "" {
				diff.WriteString(fmt.Sprintf("+%s\n", newLine))
			}
		}
	}

	return diff.String()
}

func (c *FileEditChain) generateCreateDiff(content string) string {
	lines := strings.Split(content, "\n")
	var diff strings.Builder
	diff.WriteString("--- /dev/null\n")
	diff.WriteString("+++ new\n")
	
	for _, line := range lines {
		diff.WriteString(fmt.Sprintf("+%s\n", line))
	}
	
	return diff.String()
}

func (c *FileEditChain) hashContent(content []byte) string {
	return fmt.Sprintf("%x", content[:min(8, len(content))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Chain interface methods

func (c *FileEditChain) GetMemory() schema.Memory {
	return c.memory
}

func (c *FileEditChain) GetInputKeys() []string {
	return []string{"operation", "message_id"}
}

func (c *FileEditChain) GetOutputKeys() []string {
	return []string{"result", "success", "file_context"}
}