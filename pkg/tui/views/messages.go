package views

// SwitchToPreviousViewMsg is sent when a view wants to return to the previous view
type SwitchToPreviousViewMsg struct{}

// SwitchToViewMsg is sent when switching to a specific view by index
type SwitchToViewMsg struct {
	Index int
}
