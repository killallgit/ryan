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

func TestGrepTool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GrepTool Suite")
}

var _ = Describe("GrepTool", func() {
	var (
		grepTool *tools.GrepTool
		tempDir  string
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

			properties := schema["properties"].(map[string]interface{})
			Expect(properties).To(HaveKey("pattern"))
		})
	})

	Describe("Search Operations", func() {
		It("should find matches in text files", func() {
			params := map[string]interface{}{
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
			params := map[string]interface{}{
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
			params := map[string]interface{}{
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
			params := map[string]interface{}{
				"pattern": "NonExistentPattern",
				"path":    tempDir,
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.Content).To(ContainSubstring("No matches found"))
		})

		It("should validate required parameters", func() {
			params := map[string]interface{}{
				"path": tempDir,
				// Missing pattern
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.Error).To(ContainSubstring("pattern parameter is required"))
		})

		It("should handle invalid search path", func() {
			params := map[string]interface{}{
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
			params := map[string]interface{}{
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
			params := map[string]interface{}{
				"pattern":    "main",
				"path":       tempDir,
				"file_types": []interface{}{"go"},
			}

			result, err := grepTool.Execute(context.Background(), params)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})
	})
})