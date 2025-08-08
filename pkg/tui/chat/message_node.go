package chat

func (m chatModel) createMessageNode(sender, message string) string {
	switch sender {
	case "system":
		return m.styles.SystemMessage.Render(message)
	case "user":
		return m.styles.UserMessage.Render(message)
	case "assistant":
		return m.styles.AssistantMessage.Render(message)
	case "error":
		return m.styles.ErrorMessage.Render(message)
	case "info":
		return m.styles.InfoMessage.Render(message)
	case "success":
		return m.styles.SuccessMessage.Render(message)
	default:
		return m.styles.DefaultMessage.Render(message)
	}
}

func (m chatModel) clearTextArea() {
	m.textarea.Reset()
	m.textarea.SetHeight(1)
	m.textarea.SetWidth(30)
	m.textarea.Focus()
}
