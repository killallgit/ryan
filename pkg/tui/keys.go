package tui

import "github.com/charmbracelet/bubbles/key"

type global struct {
	Quit key.Binding
	Help key.Binding
}

var keys = global{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
}
