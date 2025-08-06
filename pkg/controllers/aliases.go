package controllers

// Type aliases for cleaner naming within the package
// These provide shorter names while maintaining backward compatibility

type (
	// Shorter aliases for common use within the package
	Basic     = ChatController
	LangChain = LangChainController
	LCChat    = LangChainChatController
)

// Note: The original type names (ChatController, LangChainController, etc.)
// are preserved for backward compatibility. New code should consider using
// the shorter aliases where appropriate.
