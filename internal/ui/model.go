package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/sftp"
	"go.dev-cli.fm/internal/model"
	"go.dev-cli.fm/internal/storage"
	"golang.org/x/crypto/ssh"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	SftpClient *sftp.Client
	SshClient  *ssh.Client

	Remote model.FileManager
	Local  model.FileManager

	Focus       string // "local", "conn", "remote"
	HomeDir     string
	Connections []model.Connection

	ConnForm     ConnForm
	ConnCursor   int
	ConnStatus   string
	ConnSubFocus string // "list" или "form"

	LastKeyPress time.Time
	LastKey      string
}

func (m *Model) Init() tea.Cmd {
	return m.loadLocalDir(m.Local.Cwd)
}

func InitialModel() Model {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	startDir, err := os.Getwd()
	if err != nil {
		startDir = homeDir
	}

	localFM := model.NewFileManager()
	localFM.Cwd = startDir
	localFM.Status = ""
	localFM.CursorStack = map[string]int{startDir: 0}

	conns, err := storage.LoadConnections("connections.json")
	if err != nil {
		conns = nil
	}

	return Model{
		SftpClient:   nil,
		SshClient:    nil,
		Remote:       model.NewFileManager(),
		Local:        localFM,
		Focus:        "local",
		HomeDir:      homeDir,
		Connections:  conns,
		ConnForm:     ConnForm{Inputs: [3]string{"", "", ""}, Focused: 0, Editing: false},
		ConnCursor:   0,
		ConnSubFocus: "list",
	}
}

func InitialModelPtr() *Model {
	m := InitialModel()
	return &m
}

// Команды загрузки директорий

type filesMsg struct {
	Cwd   string
	Files []model.FileEntry
	Err   error
}

type localFilesMsg struct {
	Cwd   string
	Files []model.FileEntry
	Err   error
}

func (m *Model) loadLocalDir(path string) tea.Cmd {
	return func() tea.Msg {
		m.Local.Status = "Загрузка..."
		entries, err := os.ReadDir(path)
		if err != nil {
			return localFilesMsg{Cwd: path, Err: err}
		}
		var files []model.FileEntry
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, model.FileEntry{
				Name:  e.Name(),
				IsDir: e.IsDir(),
				Info:  info,
			})
		}
		return localFilesMsg{Cwd: path, Files: files, Err: nil}
	}
}

func (m *Model) loadRemoteDir(path string) tea.Cmd {
	return func() tea.Msg {
		if m.SftpClient == nil {
			return filesMsg{Cwd: path, Err: fmt.Errorf("SFTP client not connected")}
		}
		files, err := m.SftpClient.ReadDir(path)
		if err != nil {
			return filesMsg{Cwd: path, Err: err}
		}
		var entries []model.FileEntry
		for _, f := range files {
			entries = append(entries, model.FileEntry{
				Name:  f.Name(),
				IsDir: f.IsDir(),
				Info:  f,
			})
		}
		return filesMsg{Cwd: path, Files: entries, Err: nil}
	}
}
