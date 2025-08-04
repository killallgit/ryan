package chat

import (
	"fmt"
	"time"
)

// ConversationBranch represents a branch point in conversation history
type ConversationBranch struct {
	ID          string                 `json:"id"`
	ParentID    string                 `json:"parent_id,omitempty"`
	BranchPoint MessageID              `json:"branch_point"`
	CreatedAt   time.Time              `json:"created_at"`
	Name        string                 `json:"name,omitempty"`
	FileStates  map[string]FileContext `json:"file_states"`
	Metadata    map[string]any         `json:"metadata,omitempty"`
}

// MessageID uniquely identifies a message in the conversation
type MessageID struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Index     int       `json:"index"`
}

// FileContext tracks file state at a specific point in conversation
type FileContext struct {
	Path           string      `json:"path"`
	Content        string      `json:"content"`
	ContentHash    string      `json:"content_hash"`
	Embeddings     []float32   `json:"embeddings,omitempty"`
	LastEdit       time.Time   `json:"last_edit"`
	EditHistory    []FileEdit  `json:"edit_history"`
	MessageRefs    []MessageID `json:"message_refs"`
	ChunkRefs      []ChunkRef  `json:"chunk_refs,omitempty"`
	VectorStoreID  string      `json:"vector_store_id,omitempty"`
}

// FileEdit represents a single edit operation
type FileEdit struct {
	MessageID   MessageID `json:"message_id"`
	Timestamp   time.Time `json:"timestamp"`
	OldContent  string    `json:"old_content,omitempty"`
	NewContent  string    `json:"new_content,omitempty"`
	DiffPatch   string    `json:"diff_patch"`
	EditType    string    `json:"edit_type"` // "create", "update", "delete"
	LineChanges []LineChange `json:"line_changes,omitempty"`
}

// LineChange tracks specific line modifications
type LineChange struct {
	LineNumber int    `json:"line_number"`
	OldLine    string `json:"old_line"`
	NewLine    string `json:"new_line"`
	ChangeType string `json:"change_type"` // "add", "delete", "modify"
}

// ChunkRef references a chunk in the vector store
type ChunkRef struct {
	ChunkID    string    `json:"chunk_id"`
	StartLine  int       `json:"start_line"`
	EndLine    int       `json:"end_line"`
	Embedding  []float32 `json:"embedding,omitempty"`
	Similarity float32   `json:"similarity,omitempty"`
}

// BranchingConversation extends Conversation with branching support
type BranchingConversation struct {
	Conversation
	CurrentBranch string                       `json:"current_branch"`
	Branches      map[string]ConversationBranch `json:"branches"`
	MessageIndex  map[string]MessageID         `json:"message_index"`
}

// NewBranchingConversation creates a new branching conversation
func NewBranchingConversation(model string) *BranchingConversation {
	mainBranch := ConversationBranch{
		ID:         "main",
		CreatedAt:  time.Now(),
		Name:       "Main",
		FileStates: make(map[string]FileContext),
	}

	return &BranchingConversation{
		Conversation:  NewConversation(model),
		CurrentBranch: "main",
		Branches: map[string]ConversationBranch{
			"main": mainBranch,
		},
		MessageIndex: make(map[string]MessageID),
	}
}

// AddMessageWithContext adds a message and updates file contexts
func (bc *BranchingConversation) AddMessageWithContext(msg Message, fileContexts []FileContext) (*BranchingConversation, error) {
	// Generate message ID
	msgID := MessageID{
		ID:        generateMessageID(),
		Timestamp: msg.Timestamp,
		Index:     len(bc.Messages),
	}

	// Update message with ID in metadata
	if msg.Metadata == nil {
		msg.Metadata = &MessageMetadata{}
	}
	msg.Metadata.MessageID = msgID.ID

	// Add message to conversation
	updated := *bc
	updated.Conversation = AddMessage(bc.Conversation, msg)

	// Update message index
	updated.MessageIndex[msgID.ID] = msgID

	// Update file contexts for current branch
	branch := updated.Branches[updated.CurrentBranch]
	for _, fc := range fileContexts {
		fc.MessageRefs = append(fc.MessageRefs, msgID)
		branch.FileStates[fc.Path] = fc
	}
	updated.Branches[updated.CurrentBranch] = branch

	return &updated, nil
}

// BranchFrom creates a new branch from a specific message
func (bc *BranchingConversation) BranchFrom(messageID string, branchName string) (*BranchingConversation, error) {
	msgID, exists := bc.MessageIndex[messageID]
	if !exists {
		return nil, fmt.Errorf("message ID %s not found", messageID)
	}

	// Create new branch ID
	branchID := generateBranchID()
	if branchName == "" {
		branchName = fmt.Sprintf("Branch from %s", msgID.ID[:8])
	}

	// Copy messages up to branch point
	var branchMessages []Message
	for i := 0; i <= msgID.Index && i < len(bc.Messages); i++ {
		branchMessages = append(branchMessages, bc.Messages[i])
	}

	// Get parent branch file states at branch point
	parentBranch := bc.Branches[bc.CurrentBranch]
	branchFileStates := make(map[string]FileContext)
	
	// Copy file states that existed at branch point
	for path, context := range parentBranch.FileStates {
		// Find the file state as it was at the branch point
		relevantContext := getFileContextAtMessage(context, msgID)
		if relevantContext != nil {
			branchFileStates[path] = *relevantContext
		}
	}

	// Create new branch
	newBranch := ConversationBranch{
		ID:          branchID,
		ParentID:    bc.CurrentBranch,
		BranchPoint: msgID,
		CreatedAt:   time.Now(),
		Name:        branchName,
		FileStates:  branchFileStates,
	}

	// Create updated conversation
	updated := *bc
	updated.Messages = branchMessages
	updated.CurrentBranch = branchID
	updated.Branches[branchID] = newBranch

	return &updated, nil
}

// SwitchBranch switches to an existing branch
func (bc *BranchingConversation) SwitchBranch(branchID string) (*BranchingConversation, error) {
	_, exists := bc.Branches[branchID]
	if !exists {
		return nil, fmt.Errorf("branch %s not found", branchID)
	}

	// Reconstruct conversation for this branch
	messages := bc.getMessagesForBranch(branchID)

	updated := *bc
	updated.Messages = messages
	updated.CurrentBranch = branchID

	return &updated, nil
}

// GetFileStateAtCurrentMessage returns file state as of the latest message
func (bc *BranchingConversation) GetFileStateAtCurrentMessage(filePath string) (*FileContext, bool) {
	branch := bc.Branches[bc.CurrentBranch]
	context, exists := branch.FileStates[filePath]
	if !exists {
		return nil, false
	}
	return &context, true
}

// GetAllBranches returns all branches
func (bc *BranchingConversation) GetAllBranches() []ConversationBranch {
	branches := make([]ConversationBranch, 0, len(bc.Branches))
	for _, branch := range bc.Branches {
		branches = append(branches, branch)
	}
	return branches
}

// Helper functions

func generateMessageID() string {
	return fmt.Sprintf("msg_%d_%s", time.Now().UnixNano(), generateRandomID(8))
}

func generateBranchID() string {
	return fmt.Sprintf("branch_%d_%s", time.Now().UnixNano(), generateRandomID(8))
}

func generateRandomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func getFileContextAtMessage(fc FileContext, msgID MessageID) *FileContext {
	// Find the file state as it was at the given message
	result := FileContext{
		Path:        fc.Path,
		MessageRefs: []MessageID{},
		EditHistory: []FileEdit{},
	}

	// Include only edits up to the message timestamp
	for _, edit := range fc.EditHistory {
		if edit.Timestamp.Before(msgID.Timestamp) || edit.Timestamp.Equal(msgID.Timestamp) {
			result.EditHistory = append(result.EditHistory, edit)
			result.Content = edit.NewContent
			result.LastEdit = edit.Timestamp
		}
	}

	// Include only message refs up to this point
	for _, ref := range fc.MessageRefs {
		if ref.Timestamp.Before(msgID.Timestamp) || ref.Timestamp.Equal(msgID.Timestamp) {
			result.MessageRefs = append(result.MessageRefs, ref)
		}
	}

	if len(result.EditHistory) == 0 {
		return nil
	}

	return &result
}

func (bc *BranchingConversation) getMessagesForBranch(branchID string) []Message {
	branch, exists := bc.Branches[branchID]
	if !exists {
		return []Message{}
	}
	
	// If it's the main branch, return all messages
	if branch.ParentID == "" {
		return bc.Messages
	}

	// Otherwise, get parent messages up to branch point and continue from there
	parentMessages := bc.getMessagesForBranch(branch.ParentID)
	
	var messages []Message
	for i, msg := range parentMessages {
		messages = append(messages, msg)
		if i == branch.BranchPoint.Index {
			break
		}
	}

	// TODO: Add messages specific to this branch after branch point
	// This would require storing messages per branch

	return messages
}