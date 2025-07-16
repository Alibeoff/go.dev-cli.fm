package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"go.dev-cli.fm/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.InitialModelPtr(), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		log.Fatalf("Ошибка запуска программы: %v", err)
	}
}
