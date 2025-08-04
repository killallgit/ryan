package chat

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/vectorstore"
)

// DocumentIndexer handles automatic indexing of documents and files
type DocumentIndexer struct {
	manager         *vectorstore.Manager
	config          DocumentIndexerConfig
	log             *logger.Logger
	indexedFiles    map[string]time.Time // Track indexed files and timestamps
}

// DocumentIndexerConfig configures document indexing behavior
type DocumentIndexerConfig struct {
	CollectionName      string   // Collection to store documents
	AutoIndexFiles      bool     // Automatically index files when accessed
	AutoIndexDirectories bool    // Automatically index directory contents
	MaxFileSize         int64    // Maximum file size to index (bytes)
	SupportedExtensions []string // File extensions to index
	ExcludePatterns     []string // Patterns to exclude from indexing
	ChunkSize           int      // Size of text chunks for indexing
	UpdateInterval      time.Duration // How often to check for file updates
}

// DefaultDocumentIndexerConfig returns default configuration
func DefaultDocumentIndexerConfig() DocumentIndexerConfig {
	return DocumentIndexerConfig{
		CollectionName:      "documents",
		AutoIndexFiles:      true,
		AutoIndexDirectories: false, // Manual control for directories
		MaxFileSize:         10 * 1024 * 1024, // 10MB limit
		SupportedExtensions: []string{
			".txt", ".md", ".py", ".go", ".js", ".ts", ".json", ".yaml", ".yml",
			".html", ".css", ".sql", ".sh", ".bat", ".toml", ".ini", ".conf",
			".log", ".csv", ".xml", ".java", ".c", ".cpp", ".h", ".hpp",
			".php", ".rb", ".rs", ".swift", ".kt", ".scala", ".clj", ".r",
		},
		ExcludePatterns: []string{
			"node_modules", ".git", ".svn", "vendor", "__pycache__", ".pytest_cache",
			"build", "dist", "target", "bin", "obj", ".vscode", ".idea",
			"*.min.js", "*.min.css", "*.bundle.js", "*.lock", "*.sum",
		},
		ChunkSize:      1000,
		UpdateInterval: 5 * time.Minute,
	}
}

// NewDocumentIndexer creates a new document indexer
func NewDocumentIndexer(manager *vectorstore.Manager, config DocumentIndexerConfig) *DocumentIndexer {
	return &DocumentIndexer{
		manager:      manager,
		config:       config,
		log:          logger.WithComponent("document_indexer"),
		indexedFiles: make(map[string]time.Time),
	}
}

// shouldIndexFile determines if a file should be indexed
func (di *DocumentIndexer) shouldIndexFile(filePath string, info fs.FileInfo) bool {
	// Check file size
	if info.Size() > di.config.MaxFileSize {
		return false
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return false
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(filePath))
	validExtension := false
	for _, allowedExt := range di.config.SupportedExtensions {
		if ext == allowedExt {
			validExtension = true
			break
		}
	}
	if !validExtension {
		return false
	}

	// Check exclude patterns
	for _, pattern := range di.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
			return false
		}
		if strings.Contains(filePath, pattern) {
			return false
		}
	}

	return true
}

// IndexFile indexes a single file
func (di *DocumentIndexer) IndexFile(ctx context.Context, filePath string) error {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// Check if file should be indexed
	if !di.shouldIndexFile(filePath, info) {
		return fmt.Errorf("file %s should not be indexed", filePath)
	}

	// Check if file has been modified since last indexing
	if lastIndexed, exists := di.indexedFiles[filePath]; exists {
		if info.ModTime().Before(lastIndexed.Add(di.config.UpdateInterval)) {
			di.log.Debug("File not modified since last index", "file", filePath)
			return nil
		}
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Create base document
	doc := vectorstore.Document{
		ID:      fmt.Sprintf("file_%s_%d", strings.ReplaceAll(filePath, "/", "_"), info.ModTime().Unix()),
		Content: string(content),
		Metadata: map[string]interface{}{
			"type":          "file",
			"file_path":     filePath,
			"file_name":     filepath.Base(filePath),
			"file_ext":      filepath.Ext(filePath),
			"file_size":     info.Size(),
			"modified_time": info.ModTime().Unix(),
			"indexed_at":    time.Now().Format(time.RFC3339),
		},
	}

	// Index the document (chunking handled by manager)
	if err := di.manager.ChunkAndIndexDocument(ctx, di.config.CollectionName, doc, nil); err != nil {
		return fmt.Errorf("failed to index file %s: %w", filePath, err)
	}

	// Track indexed file
	di.indexedFiles[filePath] = time.Now()

	di.log.Info("Successfully indexed file", "file", filePath, "size", info.Size())
	return nil
}

// IndexDirectory indexes all supported files in a directory
func (di *DocumentIndexer) IndexDirectory(ctx context.Context, dirPath string, recursive bool) error {
	var indexedCount, skippedCount int

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			di.log.Warn("Error accessing path", "path", path, "error", err)
			return nil // Continue walking
		}

		// Skip directories unless we're doing recursive indexing
		if d.IsDir() {
			if !recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Try to index the file
		if err := di.IndexFile(ctx, path); err != nil {
			di.log.Debug("Skipped file", "file", path, "reason", err.Error())
			skippedCount++
		} else {
			indexedCount++
		}

		return nil
	}

	if err := filepath.WalkDir(dirPath, walkFn); err != nil {
		return fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	di.log.Info("Directory indexing completed", 
		"directory", dirPath, 
		"indexed", indexedCount, 
		"skipped", skippedCount)

	return nil
}

// SearchDocuments searches for relevant documents
func (di *DocumentIndexer) SearchDocuments(ctx context.Context, query string, maxResults int) ([]vectorstore.Result, error) {
	return di.manager.Search(ctx, di.config.CollectionName, query, maxResults)
}

// GetFileContent retrieves the content of an indexed file
func (di *DocumentIndexer) GetFileContent(ctx context.Context, filePath string) (string, error) {
	// Search for the specific file
	query := fmt.Sprintf("file_path:%s", filePath)
	results, err := di.manager.Search(ctx, di.config.CollectionName, query, 1)
	if err != nil {
		return "", fmt.Errorf("failed to search for file: %w", err)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("file not found in index: %s", filePath)
	}

	return results[0].Document.Content, nil
}

// AutoIndexFile attempts to automatically index a file if auto-indexing is enabled
func (di *DocumentIndexer) AutoIndexFile(ctx context.Context, filePath string) {
	if !di.config.AutoIndexFiles {
		return
	}

	// Run indexing in background to avoid blocking
	go func() {
		if err := di.IndexFile(ctx, filePath); err != nil {
			di.log.Debug("Auto-indexing failed", "file", filePath, "error", err)
		}
	}()
}

// GetIndexedFiles returns a list of currently indexed files
func (di *DocumentIndexer) GetIndexedFiles() []string {
	files := make([]string, 0, len(di.indexedFiles))
	for file := range di.indexedFiles {
		files = append(files, file)
	}
	return files
}

// ClearIndex removes all indexed documents
func (di *DocumentIndexer) ClearIndex(ctx context.Context) error {
	if err := di.manager.ClearCollection(ctx, di.config.CollectionName); err != nil {
		return fmt.Errorf("failed to clear document index: %w", err)
	}

	// Clear tracking map
	di.indexedFiles = make(map[string]time.Time)

	di.log.Info("Document index cleared")
	return nil
}

// RefreshIndex re-indexes all tracked files
func (di *DocumentIndexer) RefreshIndex(ctx context.Context) error {
	var refreshed, failed int

	for filePath := range di.indexedFiles {
		if err := di.IndexFile(ctx, filePath); err != nil {
			di.log.Warn("Failed to refresh file", "file", filePath, "error", err)
			failed++
		} else {
			refreshed++
		}
	}

	di.log.Info("Index refresh completed", "refreshed", refreshed, "failed", failed)
	return nil
}