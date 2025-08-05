package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASTTool_Basic(t *testing.T) {
	tool := NewASTTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "ast_parse", tool.Name())
	assert.Contains(t, tool.Description(), "Abstract Syntax Trees")
}

func TestASTTool_JSONSchema(t *testing.T) {
	tool := NewASTTool()
	schema := tool.JSONSchema()

	assert.NotNil(t, schema)
	assert.Contains(t, schema, "properties")

	properties := schema["properties"].(map[string]any)
	assert.Contains(t, properties, "file_path")
	assert.Contains(t, properties, "language")
	assert.Contains(t, properties, "analysis_type")
}

func TestASTTool_LanguageDetection(t *testing.T) {
	tool := NewASTTool()

	testCases := []struct {
		filePath string
		expected string
	}{
		{"test.go", "go"},
		{"test.py", "python"},
		{"test.js", "javascript"},
		{"test.ts", "typescript"},
		{"test.java", "java"},
		{"test.c", "c"},
		{"test.cpp", "cpp"},
		{"test.rs", "rust"},
		{"test.php", "php"},
		{"test.unknown", "unknown"},
	}

	for _, tc := range testCases {
		result := tool.detectLanguage(tc.filePath)
		assert.Equal(t, tc.expected, result, "Failed for %s", tc.filePath)
	}
}

func TestASTTool_SupportedLanguages(t *testing.T) {
	tool := NewASTTool()

	// Test supported languages
	assert.True(t, tool.isLanguageSupported("go"))
	assert.True(t, tool.isLanguageSupported("python"))
	assert.True(t, tool.isLanguageSupported("javascript"))

	// Test unsupported language
	assert.False(t, tool.isLanguageSupported("unknown"))
	assert.False(t, tool.isLanguageSupported(""))
}

func TestASTTool_ExecuteInvalidParams(t *testing.T) {
	tool := NewASTTool()
	ctx := context.Background()

	// Missing file_path
	result, err := tool.Execute(ctx, map[string]any{})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "file_path parameter is required")

	// Empty file_path
	result, err = tool.Execute(ctx, map[string]any{
		"file_path": "",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "file_path parameter is required")
}

func TestASTTool_UnsupportedLanguage(t *testing.T) {
	tool := NewASTTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"file_path": "test.unknown",
		"language":  "unsupported",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "unsupported language")
}

func TestASTTool_GoAnalysisPlaceholder(t *testing.T) {
	tool := NewASTTool()
	ctx := context.Background()

	// Create a temporary Go file
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "test.go")
	goCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func add(a, b int) int {
	return a + b
}

type Person struct {
	Name string
	Age  int
}
`

	err := os.WriteFile(goFile, []byte(goCode), 0644)
	require.NoError(t, err)

	// Test with current placeholder implementation
	result, err := tool.Execute(ctx, map[string]any{
		"file_path":     goFile,
		"language":      "go",
		"analysis_type": "full",
	})

	// Should fail with readFile not implemented
	require.NoError(t, err)
	assert.False(t, result.Success)
	// The error will be from the readFile method or Go parser
}

func TestASTTool_PythonNotImplemented(t *testing.T) {
	tool := NewASTTool()
	ctx := context.Background()

	// Create a temporary Python file
	tmpDir := t.TempDir()
	pyFile := filepath.Join(tmpDir, "test.py")
	pyCode := `def hello():
    print("Hello, World!")

class Person:
    def __init__(self, name):
        self.name = name
`

	err := os.WriteFile(pyFile, []byte(pyCode), 0644)
	require.NoError(t, err)

	result, err := tool.Execute(ctx, map[string]any{
		"file_path": pyFile,
		"language":  "python",
	})

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "Python AST analysis not implemented yet")
}

func TestASTTool_JSNotImplemented(t *testing.T) {
	tool := NewASTTool()
	ctx := context.Background()

	// Create a temporary JS file
	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "test.js")
	jsCode := `function hello() {
    console.log("Hello, World!");
}

class Person {
    constructor(name) {
        this.name = name;
    }
}
`

	err := os.WriteFile(jsFile, []byte(jsCode), 0644)
	require.NoError(t, err)

	result, err := tool.Execute(ctx, map[string]any{
		"file_path": jsFile,
		"language":  "javascript",
	})

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "JavaScript/TypeScript AST analysis not implemented yet")
}

func TestASTTool_AnalysisTypes(t *testing.T) {
	tool := NewASTTool()

	validTypes := []string{"structure", "symbols", "metrics", "issues", "full"}

	for _, analysisType := range validTypes {
		result, err := tool.Execute(context.Background(), map[string]any{
			"file_path":     "nonexistent.go",
			"analysis_type": analysisType,
		})

		require.NoError(t, err)
		// Should fail on file validation or parsing, not on analysis type
		assert.False(t, result.Success)
		assert.NotContains(t, result.Error, "invalid analysis type")
	}
}

func TestASTTool_ParameterHandling(t *testing.T) {
	tool := NewASTTool()
	ctx := context.Background()

	// Test with all optional parameters
	result, err := tool.Execute(ctx, map[string]any{
		"file_path":        "test.go",
		"language":         "go",
		"analysis_type":    "structure",
		"include_children": false,
		"max_depth":        25.0, // JSON numbers are float64
	})

	require.NoError(t, err)
	// Should fail on file validation, but parameters should be processed correctly
	assert.False(t, result.Success)
}

func TestASTTool_ResultMetadata(t *testing.T) {
	tool := NewASTTool()
	ctx := context.Background()

	start := time.Now()
	result, err := tool.Execute(ctx, map[string]any{
		"file_path": "nonexistent.go",
	})
	end := time.Now()

	require.NoError(t, err)
	assert.False(t, result.Success)

	// Check metadata
	assert.Equal(t, "ast_parse", result.Metadata.ToolName)
	assert.True(t, result.Metadata.StartTime.After(start.Add(-time.Second)))
	assert.True(t, result.Metadata.EndTime.Before(end.Add(time.Second)))
	assert.True(t, result.Metadata.ExecutionTime > 0)
}

func TestASTTool_GoHelperMethods(t *testing.T) {
	tool := NewASTTool()

	// Test visibility detection
	assert.Equal(t, "public", tool.getGoVisibility("PublicFunction"))
	assert.Equal(t, "private", tool.getGoVisibility("privateFunction"))
	assert.Equal(t, "private", tool.getGoVisibility("_internal"))
}

func TestASTTool_FormatResult(t *testing.T) {
	tool := NewASTTool()

	result := &ASTAnalysisResult{
		Language:  "go",
		FilePath:  "test.go",
		ParseTime: 10 * time.Millisecond,
		AST: ASTNode{
			Type:     "File",
			Children: make([]ASTNode, 3),
		},
		Symbols: []Symbol{
			{
				Name:     "main",
				Type:     "function",
				Kind:     "private",
				Position: ASTPosition{Line: 3},
			},
		},
		Metrics: ASTMetrics{
			Functions:  2,
			Classes:    1,
			Variables:  3,
			Complexity: 5,
			MaxNesting: 3,
		},
		Issues: []ASTIssue{
			{
				Type:     "warning",
				Category: "complexity",
				Message:  "Function too long",
				Position: ASTPosition{Line: 10},
				Severity: "medium",
			},
		},
		Dependencies: []string{"fmt", "os"},
	}

	// Test different analysis types
	analysisTypes := []string{"structure", "symbols", "metrics", "issues", "full"}

	for _, analysisType := range analysisTypes {
		output := tool.formatResult(result, analysisType)

		assert.Contains(t, output, "AST Analysis Results")
		assert.Contains(t, output, "test.go")
		assert.Contains(t, output, "10ms")

		switch analysisType {
		case "structure":
			assert.Contains(t, output, "AST Structure")
			assert.Contains(t, output, "Children: 3")
		case "symbols":
			assert.Contains(t, output, "Symbols (1)")
			assert.Contains(t, output, "function main")
		case "metrics":
			assert.Contains(t, output, "Code Metrics")
			assert.Contains(t, output, "Functions: 2")
		case "issues":
			assert.Contains(t, output, "Issues (1)")
			assert.Contains(t, output, "Function too long")
		case "full":
			assert.Contains(t, output, "AST Structure")
			assert.Contains(t, output, "Symbols (1)")
			assert.Contains(t, output, "Code Metrics")
			assert.Contains(t, output, "Issues (1)")
		}

		if analysisType == "full" {
			assert.Contains(t, output, "Dependencies (2)")
			assert.Contains(t, output, "fmt")
			assert.Contains(t, output, "os")
		}
	}
}

func TestASTNode_Structure(t *testing.T) {
	node := ASTNode{
		Type: "FuncDecl",
		Name: "testFunc",
		Position: ASTPosition{
			File:   "test.go",
			Line:   10,
			Column: 5,
			Offset: 100,
		},
		Properties: map[string]interface{}{
			"exported": true,
			"params":   2,
		},
		Children: []ASTNode{
			{Type: "Ident", Name: "param1"},
			{Type: "Ident", Name: "param2"},
		},
	}

	assert.Equal(t, "FuncDecl", node.Type)
	assert.Equal(t, "testFunc", node.Name)
	assert.Equal(t, 10, node.Position.Line)
	assert.Equal(t, true, node.Properties["exported"])
	assert.Equal(t, 2, len(node.Children))
}

func TestSymbol_Structure(t *testing.T) {
	symbol := Symbol{
		Name:       "testFunction",
		Type:       "function",
		Kind:       "public",
		Position:   ASTPosition{Line: 5, Column: 1},
		Signature:  "testFunction(a int, b string) error",
		References: 3,
	}

	assert.Equal(t, "testFunction", symbol.Name)
	assert.Equal(t, "function", symbol.Type)
	assert.Equal(t, "public", symbol.Kind)
	assert.Equal(t, 5, symbol.Position.Line)
	assert.Contains(t, symbol.Signature, "testFunction")
	assert.Equal(t, 3, symbol.References)
}

func TestASTIssue_Structure(t *testing.T) {
	issue := ASTIssue{
		Type:       "warning",
		Category:   "complexity",
		Message:    "Function is too complex",
		Position:   ASTPosition{Line: 15, Column: 1},
		Severity:   "high",
		Suggestion: "Break into smaller functions",
	}

	assert.Equal(t, "warning", issue.Type)
	assert.Equal(t, "complexity", issue.Category)
	assert.Contains(t, issue.Message, "complex")
	assert.Equal(t, "high", issue.Severity)
	assert.Contains(t, issue.Suggestion, "smaller")
}
