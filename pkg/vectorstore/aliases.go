package vectorstore

// Type aliases for cleaner naming within the package
// These provide shorter names while maintaining backward compatibility

type (
	// Interface aliases without redundant names
	Store = VectorStore // VectorStore interface -> Store

	// Type aliases for cleaner naming
	Indexer   = DocumentIndexer   // DocumentIndexer -> Indexer
	Processor = DocumentProcessor // DocumentProcessor -> Processor
)

// Note: The original type names (VectorStore, DocumentIndexer, DocumentProcessor)
// are preserved for backward compatibility. New code should consider using
// the shorter aliases where appropriate.
