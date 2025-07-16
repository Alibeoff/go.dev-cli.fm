package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"go.dev-cli.fm/internal/model"
)

var (
	borderColor       = lipgloss.Color("#7D56F4")
	remoteBorderColor = lipgloss.Color("#9B59B6")
	connBorderColor   = borderColor
	cursorColor       = lipgloss.Color("#7D56F4")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(borderColor).
			Padding(0, 1).
			MarginBottom(1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(cursorColor).
			Bold(true)

	statusHeight = 3 // Кол-во строк для вывода статуса под окнами
)

// View вызывается bubbletea, делегируем в RenderUI
func (m *Model) View() string {
	return m.RenderUI()
}

func (m *Model) RenderLocalFM() string {
	localBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(model.ListWidth).
		Height(model.ListHeight)

	return m.Local.Render(m.Focus == "local", "Локальный ФМ", localBorderStyle, borderColor)
}

func (m *Model) RenderRemoteFM() string {
	remoteBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(remoteBorderColor).
		Padding(0, 1).
		Width(model.ListWidth).
		Height(model.ListHeight)

	return m.Remote.Render(m.Focus == "remote", "SFTP ФМ", remoteBorderStyle, remoteBorderColor)
}

func (m *Model) RenderConnections() string {
	var b strings.Builder

	b.WriteString("Сохранённые соединения:\n\n")
	if len(m.Connections) == 0 {
		b.WriteString("  (нет сохранённых серверов)\n\n")
	} else {
		for i, c := range m.Connections {
			cursor := "  "
			if i == m.ConnCursor && m.Focus == "conn" && m.ConnSubFocus == "list" {
				cursor = cursorStyle.Render("▶ ")
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, c.Name))
		}
	}
	b.WriteString("\n")
	if m.ConnForm.Editing && m.ConnSubFocus == "form" {
		b.WriteString("Добавить новое соединение (Ctrl+A - сохранить, Ctrl+C - отмена):\n")
		labels := []string{"Имя: ", "Host: ", "Путь к ключу: "}
		for i, label := range labels {
			line := label + m.ConnForm.Inputs[i]
			if m.ConnForm.Focused == i {
				line = cursorStyle.Render(line + "_")
			}
			b.WriteString(line + "\n")
		}
	} else {
		b.WriteString("Нажмите 'c' для добавления нового соединения\n")
	}
	return b.String()
}

// новая функция — возвращает текст для статуса
func (m *Model) getStatusText() string {
	// Можно менять логику, сейчас выводим статус из текущего фокуса
	switch m.Focus {
	case "local":
		if m.Local.Status != "" {
			return m.Local.Status
		}
	case "remote":
		if m.Remote.Status != "" {
			return m.Remote.Status
		}
	case "conn":
		if m.ConnStatus != "" {
			return m.ConnStatus
		}
	}
	return "" // пустая строка, если нет статуса
}

func (m *Model) RenderUI() string {
	localView := m.RenderLocalFM()

	connTitleColor := lipgloss.Color("#888888")
	if m.Focus == "conn" {
		connTitleColor = connBorderColor
	}
	connTitle := titleStyle.Copy().Foreground(connTitleColor).Render(" Connection ФМ ")

	connBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(connBorderColor).
		Padding(0, 1).
		Width(model.ListWidth).
		Height(model.ListHeight)

	connView := lipgloss.JoinVertical(
		lipgloss.Left,
		connTitle,
		connBorderStyle.Render(m.RenderConnections()),
	)

	var content string
	if m.SftpClient == nil {
		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			localView,
			lipgloss.NewStyle().Width(4).Render(""),
			connView,
		)
	} else {
		remoteView := m.RenderRemoteFM()

		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			localView,
			lipgloss.NewStyle().Width(4).Render(""),
			remoteView,
			lipgloss.NewStyle().Width(4).Render(""),
			connView,
		)
	}

	// Размер статусного блока
	statusHeight := 3
	statusWidth := (model.ListWidth * 3) + 8 // приближённо ширина трёх окон + отступы

	// Получаем статус для текущего фокуса
	statusText := m.getStatusText()
	if statusText == "" {
		statusText = " " // чтобы блок всегда занимал место и не схлопывался
	}

	statusStyled := lipgloss.NewStyle().
		Height(statusHeight).
		Width(statusWidth).
		Padding(0, 1).
		Foreground(lipgloss.Color("#FFA500")).
		Align(lipgloss.Left).
		Render(statusText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		statusStyled,
	)
}
