package model

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	ListWidth  = 32
	ListHeight = 24
)

type FileEntry struct {
	Name  string
	IsDir bool
	Info  os.FileInfo
}

type FileManager struct {
	Cwd         string
	Files       []FileEntry
	Cursor      int
	Status      string
	Initialized bool
	CursorStack map[string]int
	Scroll      int
}

func NewFileManager() FileManager {
	return FileManager{
		Cwd:         ".",
		Files:       []FileEntry{},
		Cursor:      0,
		Status:      "",
		CursorStack: map[string]int{".": 0},
	}
}

func (fm *FileManager) CalcScroll() {
	total := len(fm.Files)
	if total <= ListHeight {
		fm.Scroll = 0
		return
	}
	if fm.Cursor < 0 {
		fm.Scroll = 0
		return
	}
	if fm.Cursor >= total {
		fm.Scroll = total - ListHeight
		return
	}
	if fm.Cursor < ListHeight/2 {
		fm.Scroll = 0
		return
	}
	if fm.Cursor > total-ListHeight/2 {
		fm.Scroll = total - ListHeight
		return
	}
	fm.Scroll = fm.Cursor - ListHeight/2
}

func (fm *FileManager) SetFiles(files []FileEntry) {
	fm.Files = files
	if pos, ok := fm.CursorStack[fm.Cwd]; ok && pos < len(files) {
		fm.Cursor = pos
	} else {
		fm.Cursor = 0
	}
	fm.CalcScroll()
}

func (fm *FileManager) SaveCursor() {
	fm.CursorStack[fm.Cwd] = fm.Cursor
}

func (fm *FileManager) MoveUp() {
	if fm.Cursor > 0 {
		fm.Cursor--
		fm.CalcScroll()
	}
}

func (fm *FileManager) MoveDown() {
	if fm.Cursor < len(fm.Files)-1 {
		fm.Cursor++
		fm.CalcScroll()
	}
}

func (fm *FileManager) Render(focused bool, title string, borderStyle lipgloss.Style, focusColor lipgloss.Color) string {
	var b strings.Builder
	if len(fm.Files) == 0 {
		b.WriteString("  (пусто или загрузка...)\n")
		for i := 1; i < ListHeight; i++ {
			b.WriteString("\n")
		}
	} else {
		for i := 0; i < ListHeight; i++ {
			fileIdx := fm.Scroll + i
			if fileIdx >= len(fm.Files) {
				b.WriteString("\n")
				continue
			}
			file := fm.Files[fileIdx]
			cursor := "  "
			lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

			if fm.Cursor == fileIdx && focused {
				cursor = "▶ "
				lineStyle = lipgloss.NewStyle().Foreground(focusColor).Bold(true)
			}

			icon := "📄"
			if file.IsDir {
				icon = "📁"
				lineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true)
			}

			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, icon, lineStyle.Render(file.Name)))
		}
	}
	fileList := borderStyle.Render(b.String())

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(focusColor).Background(borderStyle.GetForeground())
	header := titleStyle.Render(" " + title + " ")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		fileList,
	)
}
