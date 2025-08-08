package tui

// // A simple program demonstrating the text area component from the Bubbles
// // component library.

// import (
// 	"fmt"
// 	"strings"

// 	"github.com/charmbracelet/bubbles/textarea"
// 	"github.com/charmbracelet/bubbles/viewport"
// 	tea "github.com/charmbracelet/bubbletea"
// 	"github.com/charmbracelet/lipgloss"
// )

// const gap = "\n\n"

// type (
// 	errMsg error
// )

// type chatModel struct {
// 	viewport    viewport.Model
// 	messages    []string
// 	textarea    textarea.Model
// 	senderStyle lipgloss.Style
// 	err         error
// 	width       int
// 	height      int
// 	styles      *Styles
// }

// func NewChatModel() chatModel {
// 	ta := textarea.New()
// 	ta.Focus()
// 	ta.Placeholder = "Type a message..."
// 	ta.CharLimit = 0
// 	ta.SetHeight(1)
// 	ta.ShowLineNumbers = false
// 	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
// 	vp := viewport.New(80, 20)
// 	ta.KeyMap.InsertNewline.SetEnabled(true)

// 	styles := DefaultStyles()
// 	return chatModel{
// 		textarea:    ta,
// 		messages:    []string{},
// 		viewport:    vp,
// 		senderStyle: styles.UserMessage,
// 		styles:      styles,
// 		err:         nil,
// 	}
// }

// func (m chatModel) Init() tea.Cmd {
// 	return textarea.Blink
// }

// func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	var (
// 		tiCmd tea.Cmd
// 		vpCmd tea.Cmd
// 	)

// 	m.textarea, tiCmd = m.textarea.Update(msg)
// 	m.viewport, vpCmd = m.viewport.Update(msg)

// 	switch msg := msg.(type) {
// 	case tea.WindowSizeMsg:
// 		m.width = msg.Width
// 		m.height = msg.Height
// 		m.textarea.SetWidth(msg.Width - 4)

// 		textAreaHeight := m.calculateTextAreaHeight()
// 		m.viewport.Width = msg.Width
// 		m.viewport.Height = msg.Height - textAreaHeight - 3

// 		if len(m.messages) > 0 {
// 			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
// 		}
// 		m.viewport.GotoBottom()
// 	case tea.KeyMsg:
// 		switch msg.Type {
// 		case tea.KeyCtrlC, tea.KeyEsc:
// 			fmt.Println(m.textarea.Value())
// 			return m, tea.Quit
// 		case tea.KeyEnter:
// 			if msg.Alt {
// 				break
// 			}
// 			if m.textarea.Value() != "" {
// 				m.messages = append(m.messages, m.senderStyle.Render("You: ")+m.textarea.Value())
// 				m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
// 				m.textarea.Reset()
// 				m.textarea.SetHeight(1)
// 				m.updateViewportHeight()
// 				m.viewport.GotoBottom()
// 				return m, nil
// 			}
// 		}

// 	// We handle errors just like any other message
// 	case errMsg:
// 		m.err = msg
// 		return m, nil
// 	}

// 	prevHeight := m.textarea.Height()
// 	newHeight := m.calculateTextAreaHeight()
// 	if prevHeight != newHeight {
// 		m.textarea.SetHeight(newHeight)
// 		m.updateViewportHeight()
// 	}

// 	return m, tea.Batch(tiCmd, vpCmd)
// }

// func (m *chatModel) calculateTextAreaHeight() int {
// 	lines := strings.Count(m.textarea.Value(), "\n") + 1
// 	maxHeight := 10
// 	if lines > maxHeight {
// 		return maxHeight
// 	}
// 	if lines < 1 {
// 		return 1
// 	}
// 	return lines
// }

// func (m *chatModel) updateViewportHeight() {
// 	if m.height > 0 {
// 		textAreaHeight := m.calculateTextAreaHeight()
// 		m.viewport.Height = m.height - textAreaHeight - 3
// 	}
// }
