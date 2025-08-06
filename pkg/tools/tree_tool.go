package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// TreeTool implements directory operations with advanced filtering and analysis
type TreeTool struct {
	log            *logger.Logger
	maxDepth       int
	maxFiles       int
	maxTotalSize   int64
	followSymlinks bool
	showHidden     bool
	workingDir     string
}

// TreeEntry represents a file or directory in the tree
type TreeEntry struct {
	Name      string            `json:"name"`
	Path      string            `json:"path"`
	Type      string            `json:"type"` // "file", "dir", "symlink"
	Size      int64             `json:"size"`
	Mode      string            `json:"mode"`
	ModTime   time.Time         `json:"mod_time"`
	IsHidden  bool              `json:"is_hidden"`
	Depth     int               `json:"depth"`
	Children  []*TreeEntry      `json:"children,omitempty"`
	Extension string            `json:"extension,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// TreeSummary provides statistics about the directory tree
type TreeSummary struct {
	TotalFiles       int            `json:"total_files"`
	TotalDirs        int            `json:"total_dirs"`
	TotalSize        int64          `json:"total_size"`
	MaxDepth         int            `json:"max_depth"`
	FileTypes        map[string]int `json:"file_types"`
	LargestFile      *TreeEntry     `json:"largest_file,omitempty"`
	DeepestPath      string         `json:"deepest_path"`
	HiddenFiles      int            `json:"hidden_files"`
	SymlinkCount     int            `json:"symlink_count"`
	AccessErrors     []string       `json:"access_errors,omitempty"`
	SizeDistribution map[string]int `json:"size_distribution"`
}

// NewTreeTool creates a new Tree tool with default configuration
func NewTreeTool() *TreeTool {
	wd, _ := os.Getwd()

	return &TreeTool{
		log:            logger.WithComponent("tree_tool"),
		maxDepth:       20,
		maxFiles:       10000,
		maxTotalSize:   1024 * 1024 * 1024, // 1GB limit
		followSymlinks: false,
		showHidden:     false,
		workingDir:     wd,
	}
}

// Name returns the tool name
func (tt *TreeTool) Name() string {
	return "tree"
}

// Description returns the tool description
func (tt *TreeTool) Description() string {
	return "Directory tree analysis tool providing comprehensive file system exploration, statistics, filtering by type/size/date, and detailed directory structure analysis for code review and project understanding."
}

// JSONSchema returns the JSON schema for the tool parameters
func (tt *TreeTool) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Directory path to analyze (defaults to current directory)",
				"default":     ".",
			},
			"max_depth": map[string]any{
				"type":        "integer",
				"description": "Maximum depth to traverse",
				"minimum":     1,
				"maximum":     50,
				"default":     10,
			},
			"include_hidden": map[string]any{
				"type":        "boolean",
				"description": "Include hidden files and directories",
				"default":     false,
			},
			"follow_symlinks": map[string]any{
				"type":        "boolean",
				"description": "Follow symbolic links",
				"default":     false,
			},
			"file_types": map[string]any{
				"type":        "array",
				"description": "Filter by file extensions (e.g., ['go', 'js', 'py'])",
				"items": map[string]any{
					"type": "string",
				},
			},
			"exclude_patterns": map[string]any{
				"type":        "array",
				"description": "Patterns to exclude (e.g., ['node_modules', '*.log', '.git'])",
				"items": map[string]any{
					"type": "string",
				},
			},
			"min_size": map[string]any{
				"type":        "integer",
				"description": "Minimum file size in bytes",
				"minimum":     0,
			},
			"max_size": map[string]any{
				"type":        "integer",
				"description": "Maximum file size in bytes",
				"minimum":     0,
			},
			"sort_by": map[string]any{
				"type":        "string",
				"description": "Sort criteria",
				"enum":        []string{"name", "size", "date", "type", "depth"},
				"default":     "name",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Output format",
				"enum":        []string{"tree", "list", "json", "summary"},
				"default":     "tree",
			},
			"max_files": map[string]any{
				"type":        "integer",
				"description": "Maximum number of files to process",
				"minimum":     1,
				"maximum":     50000,
				"default":     5000,
			},
			"show_stats": map[string]any{
				"type":        "boolean",
				"description": "Include detailed statistics",
				"default":     true,
			},
		},
		"required": []string{},
	}
}

// Execute performs the directory tree analysis
func (tt *TreeTool) Execute(ctx context.Context, params map[string]any) (ToolResult, error) {
	startTime := time.Now()

	// Extract parameters
	rootPath := tt.getStringParam(params, "path", ".")
	maxDepth := tt.getIntParam(params, "max_depth", 10)
	includeHidden := tt.getBoolParam(params, "include_hidden", false)
	followSymlinks := tt.getBoolParam(params, "follow_symlinks", false)
	fileTypes := tt.getStringArrayParam(params, "file_types")
	excludePatterns := tt.getStringArrayParam(params, "exclude_patterns")
	minSize := int64(tt.getIntParam(params, "min_size", 0))
	maxSize := int64(tt.getIntParam(params, "max_size", 0))
	sortBy := tt.getStringParam(params, "sort_by", "name")
	format := tt.getStringParam(params, "format", "tree")
	maxFiles := tt.getIntParam(params, "max_files", 5000)
	showStats := tt.getBoolParam(params, "show_stats", true)

	// Validate and resolve path
	if !filepath.IsAbs(rootPath) {
		rootPath = filepath.Join(tt.workingDir, rootPath)
	}

	if _, err := os.Stat(rootPath); err != nil {
		return tt.createErrorResult(startTime, fmt.Sprintf("path does not exist: %s", rootPath)), nil
	}

	// Validate parameters
	if maxDepth > 50 || maxDepth < 1 {
		return tt.createErrorResult(startTime, "max_depth must be between 1 and 50"), nil
	}

	if maxFiles > 50000 || maxFiles < 1 {
		return tt.createErrorResult(startTime, "max_files must be between 1 and 50000"), nil
	}

	tt.log.Debug("Starting directory tree analysis",
		"path", rootPath,
		"max_depth", maxDepth,
		"format", format,
		"max_files", maxFiles)

	// Build tree with filtering options
	options := TreeOptions{
		MaxDepth:        maxDepth,
		IncludeHidden:   includeHidden,
		FollowSymlinks:  followSymlinks,
		FileTypes:       fileTypes,
		ExcludePatterns: excludePatterns,
		MinSize:         minSize,
		MaxSize:         maxSize,
		SortBy:          sortBy,
		MaxFiles:        maxFiles,
	}

	root, summary, err := tt.buildTree(ctx, rootPath, options)
	if err != nil {
		return tt.createErrorResult(startTime, fmt.Sprintf("failed to build tree: %v", err)), nil
	}

	// Check if we hit limits
	if summary.TotalFiles >= maxFiles {
		tt.log.Warn("Hit file limit during tree traversal", "limit", maxFiles)
	}

	// Format output
	content := tt.formatOutput(root, summary, format, showStats)

	tt.log.Debug("Tree analysis completed",
		"total_files", summary.TotalFiles,
		"total_dirs", summary.TotalDirs,
		"duration", time.Since(startTime))

	return ToolResult{
		Success: true,
		Content: content,
		Data: map[string]any{
			"root_path": rootPath,
			"tree":      root,
			"summary":   summary,
			"format":    format,
			"options":   options,
		},
		Metadata: ToolMetadata{
			ExecutionTime: time.Since(startTime),
			StartTime:     startTime,
			EndTime:       time.Now(),
			ToolName:      tt.Name(),
			Parameters:    params,
		},
	}, nil
}

// TreeOptions holds configuration for tree building
type TreeOptions struct {
	MaxDepth        int
	IncludeHidden   bool
	FollowSymlinks  bool
	FileTypes       []string
	ExcludePatterns []string
	MinSize         int64
	MaxSize         int64
	SortBy          string
	MaxFiles        int
}

// buildTree constructs the directory tree with filtering and statistics
func (tt *TreeTool) buildTree(ctx context.Context, rootPath string, options TreeOptions) (*TreeEntry, *TreeSummary, error) {
	summary := &TreeSummary{
		FileTypes:        make(map[string]int),
		SizeDistribution: make(map[string]int),
		AccessErrors:     make([]string, 0),
	}

	root := &TreeEntry{
		Name:     filepath.Base(rootPath),
		Path:     rootPath,
		Type:     "dir",
		Depth:    0,
		Children: make([]*TreeEntry, 0),
		Metadata: make(map[string]string),
	}

	err := tt.traverseDirectory(ctx, root, options, summary, 0)
	if err != nil {
		return nil, nil, err
	}

	// Calculate size distribution
	tt.calculateSizeDistribution(root, summary)

	// Sort children if requested
	tt.sortTree(root, options.SortBy)

	return root, summary, nil
}

// traverseDirectory recursively builds the directory tree
func (tt *TreeTool) traverseDirectory(ctx context.Context, parent *TreeEntry, options TreeOptions, summary *TreeSummary, depth int) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check depth limit
	if depth >= options.MaxDepth {
		return nil
	}

	// Check file count limit
	if summary.TotalFiles+summary.TotalDirs >= options.MaxFiles {
		return nil
	}

	entries, err := os.ReadDir(parent.Path)
	if err != nil {
		summary.AccessErrors = append(summary.AccessErrors, fmt.Sprintf("%s: %v", parent.Path, err))
		return nil // Continue processing other directories
	}

	for _, entry := range entries {
		// Check file count limit again
		if summary.TotalFiles+summary.TotalDirs >= options.MaxFiles {
			break
		}

		name := entry.Name()
		fullPath := filepath.Join(parent.Path, name)

		// Skip hidden files if not included
		if !options.IncludeHidden && strings.HasPrefix(name, ".") {
			continue
		}

		// Check exclude patterns
		if tt.matchesExcludePattern(name, fullPath, options.ExcludePatterns) {
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			summary.AccessErrors = append(summary.AccessErrors, fmt.Sprintf("%s: %v", fullPath, err))
			continue
		}

		// Handle symlinks
		if info.Mode()&fs.ModeSymlink != 0 {
			summary.SymlinkCount++
			if !options.FollowSymlinks {
				continue
			}

			// Resolve symlink
			resolved, err := filepath.EvalSymlinks(fullPath)
			if err != nil {
				summary.AccessErrors = append(summary.AccessErrors, fmt.Sprintf("symlink %s: %v", fullPath, err))
				continue
			}

			info, err = os.Stat(resolved)
			if err != nil {
				summary.AccessErrors = append(summary.AccessErrors, fmt.Sprintf("symlink target %s: %v", resolved, err))
				continue
			}
		}

		treeEntry := &TreeEntry{
			Name:     name,
			Path:     fullPath,
			Size:     info.Size(),
			Mode:     info.Mode().String(),
			ModTime:  info.ModTime(),
			IsHidden: strings.HasPrefix(name, "."),
			Depth:    depth + 1,
			Metadata: make(map[string]string),
		}

		if info.IsDir() {
			treeEntry.Type = "dir"
			treeEntry.Children = make([]*TreeEntry, 0)
			summary.TotalDirs++

			// Update max depth
			if depth+1 > summary.MaxDepth {
				summary.MaxDepth = depth + 1
				summary.DeepestPath = fullPath
			}

			// Recursively process directory
			err := tt.traverseDirectory(ctx, treeEntry, options, summary, depth+1)
			if err != nil {
				return err
			}
		} else {
			treeEntry.Type = "file"

			// Apply file filters
			if !tt.passesFileFilters(treeEntry, options) {
				continue
			}

			summary.TotalFiles++
			summary.TotalSize += info.Size()

			// Track file extension
			ext := strings.ToLower(filepath.Ext(name))
			if ext != "" {
				ext = ext[1:] // Remove the dot
				treeEntry.Extension = ext
				summary.FileTypes[ext]++
			} else {
				summary.FileTypes["no-extension"]++
			}

			// Track largest file
			if summary.LargestFile == nil || info.Size() > summary.LargestFile.Size {
				summary.LargestFile = treeEntry
			}

			// Count hidden files
			if treeEntry.IsHidden {
				summary.HiddenFiles++
			}
		}

		parent.Children = append(parent.Children, treeEntry)
	}

	return nil
}

// passesFileFilters checks if a file passes the specified filters
func (tt *TreeTool) passesFileFilters(entry *TreeEntry, options TreeOptions) bool {
	// File type filter
	if len(options.FileTypes) > 0 {
		ext := strings.ToLower(filepath.Ext(entry.Name))
		if ext != "" {
			ext = ext[1:] // Remove the dot
		}

		found := false
		for _, allowedType := range options.FileTypes {
			if ext == strings.ToLower(allowedType) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Size filters
	if options.MinSize > 0 && entry.Size < options.MinSize {
		return false
	}

	if options.MaxSize > 0 && entry.Size > options.MaxSize {
		return false
	}

	return true
}

// matchesExcludePattern checks if a path matches any exclude pattern
func (tt *TreeTool) matchesExcludePattern(name, fullPath string, patterns []string) bool {
	for _, pattern := range patterns {
		// Simple pattern matching - could be improved with glob patterns
		if strings.Contains(name, pattern) || strings.Contains(fullPath, pattern) {
			return true
		}

		// Check if it's a full path pattern
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// calculateSizeDistribution calculates file size distribution
func (tt *TreeTool) calculateSizeDistribution(root *TreeEntry, summary *TreeSummary) {
	tt.walkTree(root, func(entry *TreeEntry) {
		if entry.Type == "file" {
			switch {
			case entry.Size < 1024: // < 1KB
				summary.SizeDistribution["<1KB"]++
			case entry.Size < 10*1024: // < 10KB
				summary.SizeDistribution["1KB-10KB"]++
			case entry.Size < 100*1024: // < 100KB
				summary.SizeDistribution["10KB-100KB"]++
			case entry.Size < 1024*1024: // < 1MB
				summary.SizeDistribution["100KB-1MB"]++
			case entry.Size < 10*1024*1024: // < 10MB
				summary.SizeDistribution["1MB-10MB"]++
			default:
				summary.SizeDistribution[">10MB"]++
			}
		}
	})
}

// walkTree walks the tree and applies a function to each entry
func (tt *TreeTool) walkTree(entry *TreeEntry, fn func(*TreeEntry)) {
	fn(entry)
	for _, child := range entry.Children {
		tt.walkTree(child, fn)
	}
}

// sortTree sorts the tree based on specified criteria
func (tt *TreeTool) sortTree(root *TreeEntry, sortBy string) {
	tt.walkTree(root, func(entry *TreeEntry) {
		if len(entry.Children) > 0 {
			sort.Slice(entry.Children, func(i, j int) bool {
				a, b := entry.Children[i], entry.Children[j]

				// Always sort directories first
				if a.Type != b.Type {
					return a.Type == "dir"
				}

				switch sortBy {
				case "size":
					return a.Size > b.Size
				case "date":
					return a.ModTime.After(b.ModTime)
				case "type":
					return a.Extension < b.Extension
				case "depth":
					return a.Depth < b.Depth
				default: // name
					return strings.ToLower(a.Name) < strings.ToLower(b.Name)
				}
			})
		}
	})
}

// formatOutput formats the tree output based on the requested format
func (tt *TreeTool) formatOutput(root *TreeEntry, summary *TreeSummary, format string, showStats bool) string {
	switch format {
	case "list":
		return tt.formatAsList(root, showStats, summary)
	case "json":
		return tt.formatAsJSON(root, summary)
	case "summary":
		return tt.formatSummary(summary)
	default: // tree
		return tt.formatAsTree(root, showStats, summary)
	}
}

// formatAsTree formats output as a traditional tree view
func (tt *TreeTool) formatAsTree(root *TreeEntry, showStats bool, summary *TreeSummary) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Directory tree for: %s\n", root.Path))
	output.WriteString(strings.Repeat("=", 50))
	output.WriteString("\n\n")

	tt.printTreeEntry(&output, root, "", true)

	if showStats {
		output.WriteString("\n")
		output.WriteString(tt.formatSummary(summary))
	}

	return output.String()
}

// printTreeEntry recursively prints tree entries with proper indentation
func (tt *TreeTool) printTreeEntry(output *strings.Builder, entry *TreeEntry, prefix string, isLast bool) {
	// Choose the appropriate tree characters
	var connector, childPrefix string
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	} else {
		connector = "├── "
		childPrefix = prefix + "│   "
	}

	// Format entry name with metadata
	name := entry.Name
	if entry.Type == "file" {
		if entry.Size > 0 {
			name = fmt.Sprintf("%s (%s)", name, tt.formatSize(entry.Size))
		}
	} else if entry.Type == "dir" {
		name = fmt.Sprintf("%s/", name)
	}

	output.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, name))

	// Print children
	for i, child := range entry.Children {
		isChildLast := i == len(entry.Children)-1
		tt.printTreeEntry(output, child, childPrefix, isChildLast)
	}
}

// formatAsList formats output as a flat list
func (tt *TreeTool) formatAsList(root *TreeEntry, showStats bool, summary *TreeSummary) string {
	var output strings.Builder
	var files []*TreeEntry

	tt.walkTree(root, func(entry *TreeEntry) {
		if entry != root {
			files = append(files, entry)
		}
	})

	output.WriteString(fmt.Sprintf("Files in: %s\n", root.Path))
	output.WriteString(strings.Repeat("=", 50))
	output.WriteString("\n\n")

	for _, file := range files {
		indent := strings.Repeat("  ", file.Depth-1)
		typeChar := "F"
		if file.Type == "dir" {
			typeChar = "D"
		}

		output.WriteString(fmt.Sprintf("%s[%s] %s", indent, typeChar, file.Name))
		if file.Type == "file" && file.Size > 0 {
			output.WriteString(fmt.Sprintf(" (%s)", tt.formatSize(file.Size)))
		}
		output.WriteString("\n")
	}

	if showStats {
		output.WriteString("\n")
		output.WriteString(tt.formatSummary(summary))
	}

	return output.String()
}

// formatAsJSON formats output as JSON (simplified representation)
func (tt *TreeTool) formatAsJSON(root *TreeEntry, summary *TreeSummary) string {
	return fmt.Sprintf("JSON output would contain tree structure and summary (simplified for display)\nTotal files: %d, Total dirs: %d, Total size: %s",
		summary.TotalFiles, summary.TotalDirs, tt.formatSize(summary.TotalSize))
}

// formatSummary formats the tree summary statistics
func (tt *TreeTool) formatSummary(summary *TreeSummary) string {
	var output strings.Builder

	output.WriteString("Directory Statistics:\n")
	output.WriteString(strings.Repeat("-", 30))
	output.WriteString("\n")
	output.WriteString(fmt.Sprintf("Total files:      %d\n", summary.TotalFiles))
	output.WriteString(fmt.Sprintf("Total directories: %d\n", summary.TotalDirs))
	output.WriteString(fmt.Sprintf("Total size:       %s\n", tt.formatSize(summary.TotalSize)))
	output.WriteString(fmt.Sprintf("Maximum depth:    %d\n", summary.MaxDepth))
	output.WriteString(fmt.Sprintf("Hidden files:     %d\n", summary.HiddenFiles))
	output.WriteString(fmt.Sprintf("Symbolic links:   %d\n", summary.SymlinkCount))

	if summary.LargestFile != nil {
		output.WriteString(fmt.Sprintf("Largest file:     %s (%s)\n",
			summary.LargestFile.Name, tt.formatSize(summary.LargestFile.Size)))
	}

	if summary.DeepestPath != "" {
		output.WriteString(fmt.Sprintf("Deepest path:     %s\n", summary.DeepestPath))
	}

	if len(summary.FileTypes) > 0 {
		output.WriteString("\nFile types:\n")
		for ext, count := range summary.FileTypes {
			output.WriteString(fmt.Sprintf("  .%s: %d\n", ext, count))
		}
	}

	if len(summary.SizeDistribution) > 0 {
		output.WriteString("\nSize distribution:\n")
		for size, count := range summary.SizeDistribution {
			output.WriteString(fmt.Sprintf("  %s: %d files\n", size, count))
		}
	}

	if len(summary.AccessErrors) > 0 {
		output.WriteString(fmt.Sprintf("\nAccess errors: %d\n", len(summary.AccessErrors)))
		for i, err := range summary.AccessErrors {
			if i < 5 { // Show first 5 errors
				output.WriteString(fmt.Sprintf("  %s\n", err))
			}
		}
		if len(summary.AccessErrors) > 5 {
			output.WriteString(fmt.Sprintf("  ... and %d more\n", len(summary.AccessErrors)-5))
		}
	}

	return output.String()
}

// formatSize formats byte size in human-readable format
func (tt *TreeTool) formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// Helper methods

func (tt *TreeTool) createErrorResult(startTime time.Time, errorMsg string) ToolResult {
	tt.log.Error("Tree tool error", "error", errorMsg)
	return ToolResult{
		Success: false,
		Content: "",
		Error:   errorMsg,
		Metadata: ToolMetadata{
			ExecutionTime: time.Since(startTime),
			StartTime:     startTime,
			EndTime:       time.Now(),
			ToolName:      tt.Name(),
		},
	}
}

func (tt *TreeTool) getStringParam(params map[string]any, key, defaultValue string) string {
	if value, exists := params[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

func (tt *TreeTool) getBoolParam(params map[string]any, key string, defaultValue bool) bool {
	if value, exists := params[key]; exists {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	return defaultValue
}

func (tt *TreeTool) getIntParam(params map[string]any, key string, defaultValue int) int {
	if value, exists := params[key]; exists {
		if intValue, ok := value.(int); ok {
			return intValue
		}
		if floatValue, ok := value.(float64); ok {
			return int(floatValue)
		}
	}
	return defaultValue
}

func (tt *TreeTool) getStringArrayParam(params map[string]any, key string) []string {
	if value, exists := params[key]; exists {
		if arrayValue, ok := value.([]any); ok {
			var result []string
			for _, item := range arrayValue {
				if strValue, ok := item.(string); ok {
					result = append(result, strValue)
				}
			}
			return result
		}
	}
	return []string{}
}
