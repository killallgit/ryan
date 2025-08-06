package tools

// Type aliases for cleaner naming within the package
// These provide shorter names while maintaining backward compatibility

type (
	// Tool aliases without redundant "Tool" suffix
	Bash     = BashTool
	FileRead = FileReadTool
	Write    = WriteTool
	Grep     = GrepTool
	Git      = GitTool
	Tree     = TreeTool
	AST      = ASTTool
	WebFetch = WebFetchTool
)

// Note: The original type names (BashTool, FileReadTool, etc.)
// are preserved for backward compatibility. New code should consider using
// the shorter aliases where appropriate.
