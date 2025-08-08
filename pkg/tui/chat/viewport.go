package chat

import (
	"github.com/charmbracelet/bubbles/viewport"
)

func createViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	return vp
}
