package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"go.dev-cli.fm/internal/model"
	"go.dev-cli.fm/internal/storage"
)

type ConnForm struct {
	Inputs  [3]string
	Focused int
	Editing bool
}

func (m *Model) HandleConnFormKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.ConnForm = ConnForm{Inputs: [3]string{"", "", ""}, Focused: 0, Editing: false}
		m.ConnSubFocus = "list"

	case "ctrl+a":
		newConn := model.Connection{
			Name: m.ConnForm.Inputs[0],
			Host: m.ConnForm.Inputs[1],
			Path: m.ConnForm.Inputs[2],
		}
		if newConn.Name != "" && newConn.Host != "" && newConn.Path != "" {
			m.Connections = append(m.Connections, newConn)
			_ = storage.SaveConnections("connections.json", m.Connections)
		}
		m.ConnForm = ConnForm{Inputs: [3]string{"", "", ""}, Focused: 0, Editing: false}
		m.ConnCursor = len(m.Connections) - 1
		m.ConnSubFocus = "list"

	case "enter":
		m.ConnForm.Focused = (m.ConnForm.Focused + 1) % 3

	default:
		if len(msg.Runes) > 0 {
			r := msg.Runes[0]
			m.ConnForm.Inputs[m.ConnForm.Focused] += string(r)
		} else if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
			input := m.ConnForm.Inputs[m.ConnForm.Focused]
			if len(input) > 0 {
				m.ConnForm.Inputs[m.ConnForm.Focused] = input[:len(input)-1]
			}
		}
	}
}
