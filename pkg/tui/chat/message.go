package chat

func (m chatModel) createMessageNode(message string) string {
	return m.senderStyle.Render("You: ") + message
}
