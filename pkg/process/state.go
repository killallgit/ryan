package process

// State represents the current processing state of the system
type State string

const (
	// StateIdle indicates no active processing
	StateIdle State = ""

	// StateSending indicates data is being sent to the LLM
	StateSending State = "sending"

	// StateReceiving indicates data is being received from the LLM
	StateReceiving State = "receiving"

	// StateThinking indicates the LLM is processing/thinking
	StateThinking State = "thinking"

	// StateToolUse indicates a tool is being executed
	StateToolUse State = "tool"
)

// String returns the string representation of the state
func (s State) String() string {
	return string(s)
}

// GetIcon returns the appropriate icon for a given process state
func (s State) GetIcon() string {
	switch s {
	case StateSending:
		return "↑"
	case StateReceiving:
		return "↓"
	case StateToolUse:
		return "🔨"
	case StateThinking:
		return "🤔"
	default:
		return ""
	}
}

// GetDisplayName returns a human-readable name for the state
func (s State) GetDisplayName() string {
	switch s {
	case StateSending:
		return "Sending"
	case StateReceiving:
		return "Receiving"
	case StateThinking:
		return "Thinking"
	case StateToolUse:
		return "Using tools"
	case StateIdle:
		return "Idle"
	default:
		return ""
	}
}
