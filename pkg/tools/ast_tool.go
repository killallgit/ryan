package tools

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// ASTTool provides language-specific parsing and analysis capabilities for code review
type ASTTool struct {
	log            *logger.Logger
	supportedLangs []string
	maxFileSize    int64
	maxDepth       int
	allowedPaths   []string
	workingDir     string
}

// ASTNode represents a parsed AST node with metadata
type ASTNode struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name,omitempty"`
	Position   ASTPosition            `json:"position"`
	Children   []ASTNode              `json:"children,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ASTPosition represents source position information
type ASTPosition struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Offset int    `json:"offset"`
}

// ASTAnalysisResult contains the parsed AST and analysis metadata
type ASTAnalysisResult struct {
	Language     string        `json:"language"`
	FilePath     string        `json:"file_path"`
	AST          ASTNode       `json:"ast"`
	Symbols      []Symbol      `json:"symbols"`
	Dependencies []string      `json:"dependencies"`
	Metrics      ASTMetrics    `json:"metrics"`
	Issues       []ASTIssue    `json:"issues,omitempty"`
	ParseTime    time.Duration `json:"parse_time"`
}

// Symbol represents a code symbol (function, variable, type, etc.)
type Symbol struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"` // function, variable, type, etc.
	Kind       string      `json:"kind"` // public, private, etc.
	Position   ASTPosition `json:"position"`
	Signature  string      `json:"signature,omitempty"`
	References int         `json:"references"`
}

// ASTMetrics contains code metrics extracted from AST
type ASTMetrics struct {
	Lines         int     `json:"lines"`
	Functions     int     `json:"functions"`
	Classes       int     `json:"classes"`
	Variables     int     `json:"variables"`
	Complexity    int     `json:"complexity"`
	CognitiveLoad int     `json:"cognitive_load"`
	MaxNesting    int     `json:"max_nesting"`
	Duplication   float64 `json:"duplication_ratio"`
}

// ASTIssue represents a potential code issue found during AST analysis
type ASTIssue struct {
	Type       string      `json:"type"`     // warning, error, suggestion
	Category   string      `json:"category"` // complexity, style, performance, etc.
	Message    string      `json:"message"`
	Position   ASTPosition `json:"position"`
	Severity   string      `json:"severity"` // low, medium, high, critical
	Suggestion string      `json:"suggestion,omitempty"`
}

// NewASTTool creates a new AST analysis tool
func NewASTTool() *ASTTool {
	workingDir := "."

	return &ASTTool{
		log: logger.WithComponent("ast_tool"),
		supportedLangs: []string{
			"go", "python", "javascript", "typescript",
			"java", "c", "cpp", "rust", "php",
		},
		maxFileSize: 5 * 1024 * 1024, // 5MB
		maxDepth:    50,              // Maximum AST depth
		allowedPaths: []string{
			workingDir,
		},
		workingDir: workingDir,
	}
}

// Name returns the tool name
func (at *ASTTool) Name() string {
	return "ast_parse"
}

// Description returns the tool description
func (at *ASTTool) Description() string {
	return "Parse and analyze source code using Abstract Syntax Trees (AST). Extracts code structure, symbols, metrics, and identifies potential issues."
}

// JSONSchema returns the JSON schema for the tool parameters
func (at *ASTTool) JSONSchema() map[string]any {
	schema := NewJSONSchema()

	AddProperty(schema, "file_path", JSONSchemaProperty{
		Type:        "string",
		Description: "Path to the source code file to analyze",
	})

	// Convert supportedLangs to []any for Enum
	langEnums := make([]any, len(at.supportedLangs))
	for i, lang := range at.supportedLangs {
		langEnums[i] = lang
	}

	AddProperty(schema, "language", JSONSchemaProperty{
		Type:        "string",
		Description: "Programming language (auto-detected if not specified)",
		Enum:        langEnums,
	})

	analysisTypes := []any{"structure", "symbols", "metrics", "issues", "full"}
	AddProperty(schema, "analysis_type", JSONSchemaProperty{
		Type:        "string",
		Description: "Type of analysis to perform",
		Enum:        analysisTypes,
		Default:     "full",
	})

	AddProperty(schema, "include_children", JSONSchemaProperty{
		Type:        "boolean",
		Description: "Include child nodes in AST output",
		Default:     true,
	})

	AddProperty(schema, "max_depth", JSONSchemaProperty{
		Type:        "number",
		Description: "Maximum depth for AST traversal",
		Default:     at.maxDepth,
	})

	AddRequired(schema, "file_path")

	return schema
}

// Execute performs AST parsing and analysis
func (at *ASTTool) Execute(ctx context.Context, params map[string]any) (ToolResult, error) {
	startTime := time.Now()

	// Extract parameters
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return at.createErrorResult(startTime, "file_path parameter is required"), nil
	}

	language := ""
	if lang, exists := params["language"]; exists {
		if langStr, ok := lang.(string); ok {
			language = langStr
		}
	}

	analysisType := "full"
	if aType, exists := params["analysis_type"]; exists {
		if aTypeStr, ok := aType.(string); ok {
			analysisType = aTypeStr
		}
	}

	includeChildren := true
	if inc, exists := params["include_children"]; exists {
		if incBool, ok := inc.(bool); ok {
			includeChildren = incBool
		}
	}

	maxDepth := at.maxDepth
	if md, exists := params["max_depth"]; exists {
		if mdFloat, ok := md.(float64); ok {
			maxDepth = int(mdFloat)
		}
	}

	// Validate file path
	if err := at.validatePath(filePath); err != nil {
		return at.createErrorResult(startTime, err.Error()), nil
	}

	// Auto-detect language if not specified
	if language == "" {
		language = at.detectLanguage(filePath)
	}

	if !at.isLanguageSupported(language) {
		return at.createErrorResult(startTime, fmt.Sprintf("unsupported language: %s", language)), nil
	}

	at.log.Info("Starting AST analysis",
		"file_path", filePath,
		"language", language,
		"analysis_type", analysisType)

	// Perform analysis based on language
	result, err := at.analyzeFile(ctx, filePath, language, analysisType, includeChildren, maxDepth)
	if err != nil {
		return at.createErrorResult(startTime, err.Error()), nil
	}

	endTime := time.Now()
	result.ParseTime = endTime.Sub(startTime)

	return ToolResult{
		Success: true,
		Content: at.formatResult(result, analysisType),
		Metadata: ToolMetadata{
			ExecutionTime: endTime.Sub(startTime),
			StartTime:     startTime,
			EndTime:       endTime,
			ToolName:      at.Name(),
			Parameters:    params,
		},
	}, nil
}

// analyzeFile performs the actual AST analysis based on language
func (at *ASTTool) analyzeFile(ctx context.Context, filePath, language, analysisType string, includeChildren bool, maxDepth int) (*ASTAnalysisResult, error) {
	switch language {
	case "go":
		return at.analyzeGoFile(ctx, filePath, analysisType, includeChildren, maxDepth)
	case "python":
		return at.analyzePythonFile(ctx, filePath, analysisType, includeChildren, maxDepth)
	case "javascript", "typescript":
		return at.analyzeJSFile(ctx, filePath, language, analysisType, includeChildren, maxDepth)
	default:
		return nil, fmt.Errorf("language %s not implemented yet", language)
	}
}

// analyzeGoFile analyzes Go source code
func (at *ASTTool) analyzeGoFile(ctx context.Context, filePath, analysisType string, includeChildren bool, maxDepth int) (*ASTAnalysisResult, error) {
	// Create file set for position tracking
	fset := token.NewFileSet()

	// Parse the Go file
	src, err := at.readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	result := &ASTAnalysisResult{
		Language:     "go",
		FilePath:     filePath,
		Symbols:      make([]Symbol, 0),
		Dependencies: make([]string, 0),
		Issues:       make([]ASTIssue, 0),
	}

	// Convert AST to our format
	result.AST = at.convertGoAST(fset, file, includeChildren, maxDepth, 0)

	// Extract symbols, metrics, and issues based on analysis type
	if analysisType == "symbols" || analysisType == "full" {
		result.Symbols = at.extractGoSymbols(fset, file)
	}

	if analysisType == "metrics" || analysisType == "full" {
		result.Metrics = at.calculateGoMetrics(file)
	}

	if analysisType == "issues" || analysisType == "full" {
		result.Issues = at.findGoIssues(fset, file)
	}

	// Extract dependencies
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		result.Dependencies = append(result.Dependencies, importPath)
	}

	return result, nil
}

// convertGoAST converts Go AST to our standardized format
func (at *ASTTool) convertGoAST(fset *token.FileSet, node ast.Node, includeChildren bool, maxDepth, currentDepth int) ASTNode {
	if currentDepth > maxDepth {
		return ASTNode{Type: "max_depth_reached"}
	}

	pos := fset.Position(node.Pos())
	astNode := ASTNode{
		Type: fmt.Sprintf("%T", node),
		Position: ASTPosition{
			File:   pos.Filename,
			Line:   pos.Line,
			Column: pos.Column,
			Offset: pos.Offset,
		},
		Properties: make(map[string]interface{}),
		Children:   make([]ASTNode, 0),
	}

	// Extract node-specific information
	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Name != nil {
			astNode.Name = n.Name.Name
		}
		astNode.Properties["exported"] = ast.IsExported(astNode.Name)
		if n.Type != nil && n.Type.Results != nil {
			astNode.Properties["returns"] = len(n.Type.Results.List)
		}
		if n.Type != nil && n.Type.Params != nil {
			astNode.Properties["params"] = len(n.Type.Params.List)
		}

	case *ast.GenDecl:
		astNode.Properties["token"] = n.Tok.String()

	case *ast.TypeSpec:
		if n.Name != nil {
			astNode.Name = n.Name.Name
		}
		astNode.Properties["exported"] = ast.IsExported(astNode.Name)

	case *ast.ValueSpec:
		if len(n.Names) > 0 {
			names := make([]string, len(n.Names))
			for i, name := range n.Names {
				names[i] = name.Name
			}
			astNode.Properties["names"] = names
		}

	case *ast.Ident:
		astNode.Name = n.Name
		astNode.Properties["obj"] = n.Obj != nil

	case *ast.BasicLit:
		astNode.Properties["value"] = n.Value
		astNode.Properties["kind"] = n.Kind.String()
	}

	// Recursively process children if requested
	if includeChildren && currentDepth < maxDepth {
		ast.Inspect(node, func(child ast.Node) bool {
			if child != node && child != nil {
				// Only include direct children, not all descendants
				astNode.Children = append(astNode.Children, at.convertGoAST(fset, child, false, maxDepth, currentDepth+1))
				return false // Don't traverse further
			}
			return child == node // Continue only for the root node
		})
	}

	return astNode
}

// extractGoSymbols extracts symbols from Go AST
func (at *ASTTool) extractGoSymbols(fset *token.FileSet, file *ast.File) []Symbol {
	symbols := make([]Symbol, 0)

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name != nil {
				pos := fset.Position(node.Pos())
				symbol := Symbol{
					Name: node.Name.Name,
					Type: "function",
					Kind: at.getGoVisibility(node.Name.Name),
					Position: ASTPosition{
						File:   pos.Filename,
						Line:   pos.Line,
						Column: pos.Column,
						Offset: pos.Offset,
					},
				}

				// Build function signature
				if node.Type != nil {
					symbol.Signature = at.buildGoFunctionSignature(node)
				}

				symbols = append(symbols, symbol)
			}

		case *ast.TypeSpec:
			if node.Name != nil {
				pos := fset.Position(node.Pos())
				symbol := Symbol{
					Name: node.Name.Name,
					Type: "type",
					Kind: at.getGoVisibility(node.Name.Name),
					Position: ASTPosition{
						File:   pos.Filename,
						Line:   pos.Line,
						Column: pos.Column,
						Offset: pos.Offset,
					},
				}
				symbols = append(symbols, symbol)
			}

		case *ast.ValueSpec:
			for _, name := range node.Names {
				pos := fset.Position(name.Pos())
				symbol := Symbol{
					Name: name.Name,
					Type: "variable",
					Kind: at.getGoVisibility(name.Name),
					Position: ASTPosition{
						File:   pos.Filename,
						Line:   pos.Line,
						Column: pos.Column,
						Offset: pos.Offset,
					},
				}
				symbols = append(symbols, symbol)
			}
		}
		return true
	})

	return symbols
}

// calculateGoMetrics calculates various code metrics from Go AST
func (at *ASTTool) calculateGoMetrics(file *ast.File) ASTMetrics {
	metrics := ASTMetrics{}

	ast.Inspect(file, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncDecl:
			metrics.Functions++
		case *ast.TypeSpec:
			metrics.Classes++ // Structs/interfaces count as classes
		case *ast.ValueSpec:
			metrics.Variables++
		}
		return true
	})

	// Calculate complexity (simplified cyclomatic complexity)
	metrics.Complexity = at.calculateGoComplexity(file)
	metrics.MaxNesting = at.calculateGoMaxNesting(file)

	return metrics
}

// findGoIssues identifies potential issues in Go code
func (at *ASTTool) findGoIssues(fset *token.FileSet, file *ast.File) []ASTIssue {
	issues := make([]ASTIssue, 0)

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			// Check for long functions
			if node.Body != nil {
				start := fset.Position(node.Body.Pos())
				end := fset.Position(node.Body.End())
				lines := end.Line - start.Line

				if lines > 50 {
					pos := fset.Position(node.Pos())
					issues = append(issues, ASTIssue{
						Type:     "warning",
						Category: "complexity",
						Message:  fmt.Sprintf("Function %s is too long (%d lines)", node.Name.Name, lines),
						Position: ASTPosition{
							File:   pos.Filename,
							Line:   pos.Line,
							Column: pos.Column,
							Offset: pos.Offset,
						},
						Severity:   "medium",
						Suggestion: "Consider breaking this function into smaller functions",
					})
				}
			}

			// Check for too many parameters
			if node.Type != nil && node.Type.Params != nil && len(node.Type.Params.List) > 5 {
				pos := fset.Position(node.Pos())
				issues = append(issues, ASTIssue{
					Type:     "suggestion",
					Category: "style",
					Message:  fmt.Sprintf("Function %s has too many parameters (%d)", node.Name.Name, len(node.Type.Params.List)),
					Position: ASTPosition{
						File:   pos.Filename,
						Line:   pos.Line,
						Column: pos.Column,
						Offset: pos.Offset,
					},
					Severity:   "low",
					Suggestion: "Consider using a struct to group related parameters",
				})
			}
		}
		return true
	})

	return issues
}

// Helper methods for Go analysis
func (at *ASTTool) getGoVisibility(name string) string {
	if ast.IsExported(name) {
		return "public"
	}
	return "private"
}

func (at *ASTTool) buildGoFunctionSignature(node *ast.FuncDecl) string {
	// Simplified signature building
	sig := node.Name.Name + "("
	if node.Type != nil && node.Type.Params != nil {
		sig += fmt.Sprintf("%d params", len(node.Type.Params.List))
	}
	sig += ")"
	if node.Type != nil && node.Type.Results != nil {
		sig += fmt.Sprintf(" (%d returns)", len(node.Type.Results.List))
	}
	return sig
}

func (at *ASTTool) calculateGoComplexity(file *ast.File) int {
	complexity := 1 // Base complexity

	ast.Inspect(file, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

func (at *ASTTool) calculateGoMaxNesting(file *ast.File) int {
	maxNesting := 0
	currentNesting := 0

	ast.Inspect(file, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.BlockStmt:
			currentNesting++
			if currentNesting > maxNesting {
				maxNesting = currentNesting
			}
		}
		return true
	})

	return maxNesting
}

// Placeholder implementations for other languages
func (at *ASTTool) analyzePythonFile(ctx context.Context, filePath, analysisType string, includeChildren bool, maxDepth int) (*ASTAnalysisResult, error) {
	return nil, fmt.Errorf("Python AST analysis not implemented yet")
}

func (at *ASTTool) analyzeJSFile(ctx context.Context, filePath, language, analysisType string, includeChildren bool, maxDepth int) (*ASTAnalysisResult, error) {
	return nil, fmt.Errorf("JavaScript/TypeScript AST analysis not implemented yet")
}

// Utility methods
func (at *ASTTool) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".rs":
		return "rust"
	case ".php":
		return "php"
	default:
		return "unknown"
	}
}

func (at *ASTTool) isLanguageSupported(language string) bool {
	for _, lang := range at.supportedLangs {
		if lang == language {
			return true
		}
	}
	return false
}

func (at *ASTTool) validatePath(filePath string) error {
	_, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if file exists and is readable
	// This is a simplified validation - in production you'd want more comprehensive checks
	return nil
}

func (at *ASTTool) readFile(filePath string) (string, error) {
	// This would use the FileReadTool in practice
	// For now, return empty string to satisfy interface
	return "", nil
}

func (at *ASTTool) formatResult(result *ASTAnalysisResult, analysisType string) string {
	output := fmt.Sprintf("AST Analysis Results for %s (%s)\n", result.FilePath, result.Language)
	output += fmt.Sprintf("Parse Time: %v\n\n", result.ParseTime)

	if analysisType == "structure" || analysisType == "full" {
		output += fmt.Sprintf("AST Structure:\n")
		output += fmt.Sprintf("  Root Type: %s\n", result.AST.Type)
		output += fmt.Sprintf("  Children: %d\n\n", len(result.AST.Children))
	}

	if analysisType == "symbols" || analysisType == "full" {
		output += fmt.Sprintf("Symbols (%d):\n", len(result.Symbols))
		for _, symbol := range result.Symbols {
			output += fmt.Sprintf("  %s %s (%s) at line %d\n",
				symbol.Type, symbol.Name, symbol.Kind, symbol.Position.Line)
		}
		output += "\n"
	}

	if analysisType == "metrics" || analysisType == "full" {
		output += "Code Metrics:\n"
		output += fmt.Sprintf("  Functions: %d\n", result.Metrics.Functions)
		output += fmt.Sprintf("  Types: %d\n", result.Metrics.Classes)
		output += fmt.Sprintf("  Variables: %d\n", result.Metrics.Variables)
		output += fmt.Sprintf("  Complexity: %d\n", result.Metrics.Complexity)
		output += fmt.Sprintf("  Max Nesting: %d\n\n", result.Metrics.MaxNesting)
	}

	if (analysisType == "issues" || analysisType == "full") && len(result.Issues) > 0 {
		output += fmt.Sprintf("Issues (%d):\n", len(result.Issues))
		for _, issue := range result.Issues {
			output += fmt.Sprintf("  [%s] %s: %s (line %d)\n",
				issue.Severity, issue.Category, issue.Message, issue.Position.Line)
		}
		output += "\n"
	}

	if len(result.Dependencies) > 0 {
		output += fmt.Sprintf("Dependencies (%d):\n", len(result.Dependencies))
		for _, dep := range result.Dependencies {
			output += fmt.Sprintf("  %s\n", dep)
		}
	}

	return output
}

func (at *ASTTool) createErrorResult(startTime time.Time, errorMsg string) ToolResult {
	endTime := time.Now()
	return ToolResult{
		Success: false,
		Error:   errorMsg,
		Metadata: ToolMetadata{
			ExecutionTime: endTime.Sub(startTime),
			StartTime:     startTime,
			EndTime:       endTime,
			ToolName:      at.Name(),
		},
	}
}
