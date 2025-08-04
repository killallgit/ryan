package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/killallgit/ryan/pkg/tools"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTools(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tools Suite")
}

var _ = Describe("GrepTool", func() {
	var (
		grepTool  *tools.GrepTool
		tempDir   string
		testFile1 string
		testFile2 string
	)

	BeforeEach(func() {
		grepTool = tools.NewGrepTool()

		// Create temporary directory and test files
		var err error
		tempDir, err = os.MkdirTemp("", "grep_test")
		Expect(err).ToNot(HaveOccurred())

		testFile1 = filepath.Join(tempDir, "test1.txt")
		testFile2 = filepath.Join(tempDir, "test2.go")

		// Create test files with content
		err = os.WriteFile(testFile1, []byte("Hello world\nThis is a test file\nContains some text\n"), 0644)
		Expect(err).ToNot(HaveOccurred())

		err = os.WriteFile(testFile2, []byte("package main\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n"), 0644)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("Tool Interface", func() {
		It("should have correct name", func() {
			Expect(grepTool.Name()).To(Equal("grep"))
		})

		It("should have description", func() {
			description := grepTool.Description()
			Expect(description).ToNot(BeEmpty())
			Expect(description).To(ContainSubstring("text search"))
		})

		It("should have valid JSON schema", func() {
			schema := grepTool.JSONSchema()
			Expect(schema).To(HaveKey("type"))
			Expect(schema).To(HaveKey("properties"))
			Expect(schema).To(HaveKey("required"))

			properties := schema["properties"].(map[string]any)
			Expect(properties).To(HaveKey("pattern"))
		})
	})

	Describe("Search Operations", func() {
		It("should find matches in text files", func() {
			params := map[string]any{
				"pattern": "Hello",
				"path":    tempDir,
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Content).To(ContainSubstring("Hello"))
			Expect(result.Data).To(HaveKey("results_count"))
		})

		It("should handle case insensitive search", func() {
			params := map[string]any{
				"pattern":        "HELLO",
				"path":           tempDir,
				"case_sensitive": false,
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Content).To(ContainSubstring("Hello"))
		})

		It("should return files only when requested", func() {
			params := map[string]any{
				"pattern":    "Hello",
				"path":       tempDir,
				"files_only": true,
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Content).To(ContainSubstring("Files containing matches"))
		})

		It("should handle no matches gracefully", func() {
			params := map[string]any{
				"pattern": "NonExistentPattern",
				"path":    tempDir,
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Content).To(ContainSubstring("No matches found"))
		})

		It("should validate required parameters", func() {
			params := map[string]any{
				"path": tempDir,
				// Missing pattern
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.Error).To(ContainSubstring("pattern parameter is required"))
		})

		It("should handle invalid search path", func() {
			params := map[string]any{
				"pattern": "test",
				"path":    "/nonexistent/path",
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.Error).To(ContainSubstring("search path does not exist"))
		})
	})

	Describe("Parameter Handling", func() {
		It("should use default values for optional parameters", func() {
			params := map[string]any{
				"pattern": "Hello",
				"path":    tempDir,
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Data).To(HaveKey("case_sensitive"))
			Expect(result.Data["case_sensitive"]).To(BeFalse()) // Default value
		})

		It("should respect file type filtering", func() {
			params := map[string]any{
				"pattern":    "main",
				"path":       tempDir,
				"file_types": []any{"go"},
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})
	})
})

var _ = Describe("WriteTool", func() {
	var (
		writeTool *tools.WriteTool
		tempDir   string
	)

	BeforeEach(func() {
		writeTool = tools.NewWriteTool()

		// Create temporary directory for tests
		var err error
		tempDir, err = os.MkdirTemp("", "write_test")
		Expect(err).ToNot(HaveOccurred())

		// Change to temp directory for relative path tests
		err = os.Chdir(tempDir)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("Tool Interface", func() {
		It("should have correct name", func() {
			Expect(writeTool.Name()).To(Equal("write_file"))
		})

		It("should have description", func() {
			description := writeTool.Description()
			Expect(description).ToNot(BeEmpty())
			Expect(description).To(ContainSubstring("write content"))
		})

		It("should have valid JSON schema", func() {
			schema := writeTool.JSONSchema()
			Expect(schema).To(HaveKey("type"))
			Expect(schema).To(HaveKey("properties"))
			Expect(schema).To(HaveKey("required"))

			properties := schema["properties"].(map[string]any)
			Expect(properties).To(HaveKey("file_path"))
			Expect(properties).To(HaveKey("content"))

			required := schema["required"].([]string)
			Expect(required).To(ContainElement("file_path"))
			Expect(required).To(ContainElement("content"))
		})
	})

	Describe("File Writing Operations", func() {
		It("should create a new file with content", func() {
			testFile := filepath.Join(tempDir, "new_file.txt")
			testContent := "Hello, World!\nThis is a test file."

			params := map[string]any{
				"file_path": testFile,
				"content":   testContent,
				"force":     true, // Bypass path restrictions for tests
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Content).To(ContainSubstring("Created file"))
			Expect(result.Data).To(HaveKey("bytes_written"))
			Expect(result.Data["bytes_written"]).To(Equal(len(testContent)))

			// Verify file was created with correct content
			writtenContent, err := os.ReadFile(testFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(writtenContent)).To(Equal(testContent))
		})

		It("should update existing file with backup", func() {
			testFile := filepath.Join(tempDir, "existing_file.txt")
			originalContent := "Original content"
			newContent := "Updated content"

			// Create original file
			err := os.WriteFile(testFile, []byte(originalContent), 0644)
			Expect(err).ToNot(HaveOccurred())

			params := map[string]any{
				"file_path":     testFile,
				"content":       newContent,
				"create_backup": true,
				"force":         true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Content).To(ContainSubstring("Updated file"))
			Expect(result.Content).To(ContainSubstring("Backup created"))
			Expect(result.Data["backup_created"]).To(BeTrue())

			// Verify file was updated
			writtenContent, err := os.ReadFile(testFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(writtenContent)).To(Equal(newContent))

			// Verify backup was created
			backupPath := result.Data["backup_path"].(string)
			Expect(backupPath).ToNot(BeEmpty())
			backupContent, err := os.ReadFile(backupPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(backupContent)).To(Equal(originalContent))
		})

		It("should append to existing file", func() {
			testFile := filepath.Join(tempDir, "append_file.txt")
			originalContent := "Line 1\n"
			appendContent := "Line 2\n"

			// Create original file
			err := os.WriteFile(testFile, []byte(originalContent), 0644)
			Expect(err).ToNot(HaveOccurred())

			params := map[string]any{
				"file_path": testFile,
				"content":   appendContent,
				"append":    true,
				"force":     true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Content).To(ContainSubstring("Appended to"))
			Expect(result.Data["append_mode"]).To(BeTrue())

			// Verify content was appended
			finalContent, err := os.ReadFile(testFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(finalContent)).To(Equal(originalContent + appendContent))
		})

		It("should create parent directories when requested", func() {
			nestedPath := filepath.Join(tempDir, "nested", "deep", "file.txt")
			testContent := "Nested file content"

			params := map[string]any{
				"file_path":   nestedPath,
				"content":     testContent,
				"create_dirs": true,
				"force":       true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			// Verify file and directories were created
			_, err = os.Stat(nestedPath)
			Expect(err).ToNot(HaveOccurred())

			content, err := os.ReadFile(nestedPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(testContent))
		})
	})

	Describe("Line Ending Handling", func() {
		It("should normalize line endings to LF by default", func() {
			testFile := filepath.Join(tempDir, "line_endings.txt")
			testContent := "Line 1\r\nLine 2\rLine 3\n"

			params := map[string]any{
				"file_path": testFile,
				"content":   testContent,
				"force":     true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			content, err := os.ReadFile(testFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal("Line 1\nLine 2\nLine 3\n"))
		})

		It("should handle CRLF line endings", func() {
			testFile := filepath.Join(tempDir, "crlf_file.txt")
			testContent := "Line 1\nLine 2\n"

			params := map[string]any{
				"file_path":   testFile,
				"content":     testContent,
				"line_ending": "crlf",
				"force":       true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			content, err := os.ReadFile(testFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal("Line 1\r\nLine 2\r\n"))
		})
	})

	Describe("Safety and Validation", func() {
		It("should validate required parameters", func() {
			// Missing file_path
			params := map[string]any{
				"content": "test content",
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.Error).To(ContainSubstring("file_path parameter is required"))
		})

		It("should validate content parameter", func() {
			// Missing content
			params := map[string]any{
				"file_path": "test.txt",
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.Error).To(ContainSubstring("content parameter is required"))
		})

		It("should reject restricted paths", func() {
			params := map[string]any{
				"file_path": "/etc/passwd",
				"content":   "malicious content",
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.Error).To(ContainSubstring("restricted directory"))
		})

		It("should reject unsupported file extensions", func() {
			// Use a simple filename without absolute path to avoid temp dir issues
			params := map[string]any{
				"file_path": "test.exe",
				"content":   "binary content",
				// Don't use force mode to test the validation
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.Error).To(ContainSubstring("file extension"))
		})

		It("should allow force mode to bypass safety checks", func() {
			params := map[string]any{
				"file_path": filepath.Join(tempDir, "test.exe"),
				"content":   "binary content",
				"force":     true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})

		It("should handle path traversal attempts", func() {
			// Use a path with .. that tries to access parent directories
			params := map[string]any{
				"file_path": "../../../etc/passwd",
				"content":   "malicious content",
				// Don't use force mode to test the validation
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			// The Write tool should prevent traversal attempts through various mechanisms
			// (path validation, restricted directories, or permission errors)
			Expect(result.Error).ToNot(BeEmpty())
		})
	})

	Describe("Parameter Handling", func() {
		It("should use default values for optional parameters", func() {
			testFile := filepath.Join(tempDir, "defaults.txt")

			params := map[string]any{
				"file_path": testFile,
				"content":   "test content",
				"force":     true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Data["append_mode"]).To(BeFalse())   // Default
			Expect(result.Data["encoding"]).To(Equal("utf-8")) // Default
			Expect(result.Data["line_ending"]).To(Equal("lf")) // Default
		})

		It("should respect custom parameter values", func() {
			testFile := filepath.Join(tempDir, "custom.txt")

			params := map[string]any{
				"file_path":     testFile,
				"content":       "test content",
				"create_backup": false,
				"encoding":      "ascii",
				"line_ending":   "crlf",
				"force":         true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Data["backup_created"]).To(BeFalse())
			Expect(result.Data["encoding"]).To(Equal("ascii"))
			Expect(result.Data["line_ending"]).To(Equal("crlf"))
		})
	})

	Describe("Relative and Absolute Paths", func() {
		It("should handle relative paths", func() {
			relativePath := "relative_file.txt"
			testContent := "relative path content"

			params := map[string]any{
				"file_path": relativePath,
				"content":   testContent,
				"force":     true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			// The file should be created in the current directory (tempDir)
			expectedPath := filepath.Join(tempDir, relativePath)
			_, err = os.Stat(expectedPath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle absolute paths", func() {
			absolutePath := filepath.Join(tempDir, "absolute_file.txt")
			testContent := "absolute path content"

			params := map[string]any{
				"file_path": absolutePath,
				"content":   testContent,
				"force":     true,
			}

			result, err := writeTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			_, err = os.Stat(absolutePath)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
