package agents

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// CodeAnalysisAgent performs AST analysis, symbol resolution, and code understanding
type CodeAnalysisAgent struct {
	astAnalyzer    *ASTAnalyzer
	symbolResolver *SymbolResolver
	log            *logger.Logger
}

// NewCodeAnalysisAgent creates a new code analysis agent
func NewCodeAnalysisAgent() *CodeAnalysisAgent {
	return &CodeAnalysisAgent{
		astAnalyzer:    NewASTAnalyzer(),
		symbolResolver: NewSymbolResolver(),
		log:            logger.WithComponent("code_analysis_agent"),
	}
}

// Name returns the agent name
func (c *CodeAnalysisAgent) Name() string {
	return "code_analysis"
}

// Description returns the agent description
func (c *CodeAnalysisAgent) Description() string {
	return "Performs AST analysis, symbol resolution, and code structure understanding"
}

// CanHandle determines if this agent can handle the request
func (c *CodeAnalysisAgent) CanHandle(request string) (bool, float64) {
	lowerRequest := strings.ToLower(request)
	
	keywords := []string{
		"analyze", "ast", "structure", "symbols",
		"functions", "types", "interfaces", "patterns",
	}
	
	for _, keyword := range keywords {
		if strings.Contains(lowerRequest, keyword) {
			return true, 0.8
		}
	}
	
	return false, 0.0
}

// Execute performs code analysis
func (c *CodeAnalysisAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	startTime := time.Now()
	c.log.Info("Executing code analysis", "prompt", request.Prompt)

	// Get file contents from context
	fileContents, err := c.getFileContents(request)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: "Failed to get file contents",
			Details: err.Error(),
		}, err
	}

	// Analyze each file
	analysisResults := make(map[string]*FileAnalysis)
	var allSymbols []Symbol
	var allFunctions []Function
	var allTypes []Type

	for filePath, content := range fileContents {
		if strings.HasSuffix(filePath, ".go") {
			analysis, err := c.analyzeGoFile(filePath, content)
			if err != nil {
				c.log.Warn("Failed to analyze file", "file", filePath, "error", err)
				continue
			}
			
			analysisResults[filePath] = analysis
			allSymbols = append(allSymbols, analysis.Symbols...)
			allFunctions = append(allFunctions, analysis.Functions...)
			allTypes = append(allTypes, analysis.Types...)
		}
	}

	// Build summary and details
	summary := c.buildSummary(analysisResults)
	details := c.buildDetails(analysisResults)

	return AgentResult{
		Success: true,
		Summary: summary,
		Details: details,
		Artifacts: map[string]interface{}{
			"analysis_results": analysisResults,
			"symbols":          allSymbols,
			"functions":        allFunctions,
			"types":            allTypes,
		},
		Metadata: AgentMetadata{
			AgentName:      c.Name(),
			StartTime:      startTime,
			EndTime:        time.Now(),
			Duration:       time.Since(startTime),
			FilesProcessed: getKeys(analysisResults),
		},
	}, nil
}

// getFileContents retrieves file contents from the request context
func (c *CodeAnalysisAgent) getFileContents(request AgentRequest) (map[string]string, error) {
	// Check execution context
	if execContext, ok := request.Context["execution_context"].(*ExecutionContext); ok {
		if fileContents, ok := execContext.SharedData["file_contents"].(map[string]string); ok {
			return fileContents, nil
		}
	}

	// Check direct context
	if fileContents, ok := request.Context["file_contents"].(map[string]string); ok {
		return fileContents, nil
	}

	// Check artifacts
	if artifacts, ok := request.Context["artifacts"].(map[string]interface{}); ok {
		if fileContents, ok := artifacts["file_contents"].(map[string]string); ok {
			return fileContents, nil
		}
	}

	return nil, fmt.Errorf("no file contents found in context")
}

// analyzeGoFile analyzes a single Go file
func (c *CodeAnalysisAgent) analyzeGoFile(filePath, content string) (*FileAnalysis, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	analysis := &FileAnalysis{
		FilePath:  filePath,
		Package:   node.Name.Name,
		Imports:   c.extractImports(node),
		Functions: []Function{},
		Types:     []Type{},
		Symbols:   []Symbol{},
	}

	// Walk the AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			fn := c.extractFunction(x, fset)
			analysis.Functions = append(analysis.Functions, fn)
			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name: fn.Name,
				Type: "function",
				Pos:  fn.Position,
			})
			
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						typ := c.extractType(typeSpec, fset)
						analysis.Types = append(analysis.Types, typ)
						analysis.Symbols = append(analysis.Symbols, Symbol{
							Name: typ.Name,
							Type: "type",
							Pos:  typ.Position,
						})
					}
				}
			}
		}
		return true
	})

	return analysis, nil
}

// Helper methods for extraction

func (c *CodeAnalysisAgent) extractImports(file *ast.File) []string {
	var imports []string
	for _, imp := range file.Imports {
		path := imp.Path.Value
		imports = append(imports, strings.Trim(path, `"`))
	}
	return imports
}

func (c *CodeAnalysisAgent) extractFunction(fn *ast.FuncDecl, fset *token.FileSet) Function {
	pos := fset.Position(fn.Pos())
	
	function := Function{
		Name:     fn.Name.Name,
		Position: fmt.Sprintf("%s:%d", pos.Filename, pos.Line),
		Params:   []string{},
		Returns:  []string{},
	}

	// Extract receiver
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		function.Receiver = c.typeToString(fn.Recv.List[0].Type)
	}

	// Extract parameters
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			paramType := c.typeToString(param.Type)
			for range param.Names {
				function.Params = append(function.Params, paramType)
			}
		}
	}

	// Extract return types
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			function.Returns = append(function.Returns, c.typeToString(result.Type))
		}
	}

	return function
}

func (c *CodeAnalysisAgent) extractType(spec *ast.TypeSpec, fset *token.FileSet) Type {
	pos := fset.Position(spec.Pos())
	
	typ := Type{
		Name:     spec.Name.Name,
		Position: fmt.Sprintf("%s:%d", pos.Filename, pos.Line),
	}

	// Determine type kind
	switch t := spec.Type.(type) {
	case *ast.StructType:
		typ.Kind = "struct"
		typ.Fields = c.extractStructFields(t)
	case *ast.InterfaceType:
		typ.Kind = "interface"
		typ.Methods = c.extractInterfaceMethods(t)
	default:
		typ.Kind = "alias"
	}

	return typ
}

func (c *CodeAnalysisAgent) extractStructFields(s *ast.StructType) []Field {
	var fields []Field
	for _, field := range s.Fields.List {
		fieldType := c.typeToString(field.Type)
		for _, name := range field.Names {
			fields = append(fields, Field{
				Name: name.Name,
				Type: fieldType,
			})
		}
	}
	return fields
}

func (c *CodeAnalysisAgent) extractInterfaceMethods(i *ast.InterfaceType) []Method {
	var methods []Method
	for _, method := range i.Methods.List {
		if _, ok := method.Type.(*ast.FuncType); ok {
			for _, name := range method.Names {
				methods = append(methods, Method{
					Name: name.Name,
				})
			}
		}
	}
	return methods
}

func (c *CodeAnalysisAgent) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + c.typeToString(t.X)
	case *ast.ArrayType:
		return "[]" + c.typeToString(t.Elt)
	case *ast.SelectorExpr:
		return c.typeToString(t.X) + "." + t.Sel.Name
	default:
		return "interface{}"
	}
}

// Build summary and details

func (c *CodeAnalysisAgent) buildSummary(results map[string]*FileAnalysis) string {
	totalFunctions := 0
	totalTypes := 0
	totalSymbols := 0

	for _, analysis := range results {
		totalFunctions += len(analysis.Functions)
		totalTypes += len(analysis.Types)
		totalSymbols += len(analysis.Symbols)
	}

	return fmt.Sprintf("Analyzed %d files: found %d functions, %d types, %d total symbols",
		len(results), totalFunctions, totalTypes, totalSymbols)
}

func (c *CodeAnalysisAgent) buildDetails(results map[string]*FileAnalysis) string {
	var details []string

	for filePath, analysis := range results {
		details = append(details, fmt.Sprintf("File: %s", filePath))
		details = append(details, fmt.Sprintf("  Package: %s", analysis.Package))
		
		if len(analysis.Functions) > 0 {
			details = append(details, "  Functions:")
			for _, fn := range analysis.Functions {
				sig := fn.Name
				if fn.Receiver != "" {
					sig = fmt.Sprintf("(%s) %s", fn.Receiver, fn.Name)
				}
				details = append(details, fmt.Sprintf("    - %s", sig))
			}
		}

		if len(analysis.Types) > 0 {
			details = append(details, "  Types:")
			for _, typ := range analysis.Types {
				details = append(details, fmt.Sprintf("    - %s (%s)", typ.Name, typ.Kind))
			}
		}

		details = append(details, "")
	}

	return strings.Join(details, "\n")
}

// Supporting types

type FileAnalysis struct {
	FilePath  string
	Package   string
	Imports   []string
	Functions []Function
	Types     []Type
	Symbols   []Symbol
}

type Symbol struct {
	Name string
	Type string
	Pos  string
}

type Function struct {
	Name     string
	Receiver string
	Params   []string
	Returns  []string
	Position string
}

type Type struct {
	Name     string
	Kind     string // struct, interface, alias
	Fields   []Field
	Methods  []Method
	Position string
}

type Field struct {
	Name string
	Type string
}

type Method struct {
	Name string
}

// Helper components

type ASTAnalyzer struct{}

func NewASTAnalyzer() *ASTAnalyzer {
	return &ASTAnalyzer{}
}

type SymbolResolver struct{}

func NewSymbolResolver() *SymbolResolver {
	return &SymbolResolver{}
}

// Utility function
func getKeys(m map[string]*FileAnalysis) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}