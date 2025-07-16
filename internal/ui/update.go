package ui

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/sftp"
	"go.dev-cli.fm/internal/model"
	sftpclient "go.dev-cli.fm/internal/sftp"
	"go.dev-cli.fm/internal/storage"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case filesMsg:
		if msg.Err != nil {
			m.Remote.Status = "Ошибка: " + msg.Err.Error()
			m.Remote.Files = nil
		} else {
			m.Remote.Cwd = msg.Cwd
			m.Remote.SetFiles(msg.Files)
			m.Remote.Status = ""
		}
		return m, nil

	case localFilesMsg:
		if msg.Err != nil {
			m.Local.Status = "Ошибка: " + msg.Err.Error()
			m.Local.Files = nil
		} else {
			m.Local.Cwd = msg.Cwd
			m.Local.SetFiles(msg.Files)
			m.Local.Status = ""
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case string:
		// Статусные сообщения
		if m.Focus == "local" {
			m.Local.Status = msg
		} else if m.Focus == "remote" {
			m.Remote.Status = msg
		}
		return m, nil

	case error:
		errMsg := "Ошибка: " + msg.Error()
		if m.Focus == "local" {
			m.Local.Status = errMsg
		} else if m.Focus == "remote" {
			m.Remote.Status = errMsg
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab":
		m.switchFocus()
		return m, nil
	}

	if key.String() == "d" {
		switch m.Focus {
		case "local":
			return m.deleteLocalFile()
		case "remote":
			return m.deleteRemoteFile()
		case "conn":
			if m.ConnSubFocus == "list" {
				return m.deleteConnection()
			}
		}
	}

	switch m.Focus {
	case "local":
		return m.handleLocalKey(key)
	case "remote":
		return m.handleRemoteKey(key)
	case "conn":
		if m.ConnForm.Editing && m.ConnSubFocus == "form" {
			m.HandleConnFormKey(key)
			return m, nil
		}
		m.handleConnListKey(key)
		return m, nil
	}

	return m, nil
}

func (m *Model) switchFocus() {
	switch m.Focus {
	case "local":
		m.Focus = "conn"
	case "conn":
		if m.SftpClient != nil {
			m.Focus = "remote"
		} else {
			m.Focus = "local"
		}
	case "remote":
		m.Focus = "local"
	}
}

func (m *Model) deleteLocalFile() (tea.Model, tea.Cmd) {
	if len(m.Local.Files) == 0 {
		return m, nil
	}
	file := m.Local.Files[m.Local.Cursor]
	path := filepath.Join(m.Local.Cwd, file.Name)
	err := os.RemoveAll(path)
	if err != nil {
		m.Local.Status = "Ошибка удаления: " + err.Error()
		return m, nil
	}
	m.Local.Status = "Файл удалён: " + file.Name
	return m, m.loadLocalDir(m.Local.Cwd)
}

func (m *Model) deleteRemoteFile() (tea.Model, tea.Cmd) {
	if m.SftpClient == nil || len(m.Remote.Files) == 0 {
		return m, nil
	}
	file := m.Remote.Files[m.Remote.Cursor]
	path := filepath.Join(m.Remote.Cwd, file.Name)
	err := m.SftpClient.Remove(path)
	if err != nil {
		m.Remote.Status = "Ошибка удаления: " + err.Error()
		return m, nil
	}
	m.Remote.Status = "Файл удалён: " + file.Name
	return m, m.loadRemoteDir(m.Remote.Cwd)
}

func (m *Model) deleteConnection() (tea.Model, tea.Cmd) {
	if len(m.Connections) == 0 {
		return m, nil
	}
	m.Connections = append(m.Connections[:m.ConnCursor], m.Connections[m.ConnCursor+1:]...)
	if m.ConnCursor >= len(m.Connections) && m.ConnCursor > 0 {
		m.ConnCursor--
	}
	_ = storage.SaveConnections("connections.json", m.Connections)
	return m, nil
}

func (m *Model) handleLocalKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "k", "up":
		m.Local.MoveUp()
	case "j", "down":
		m.Local.MoveDown()
	case "h", "backspace":
		parent := filepath.Dir(m.Local.Cwd)
		if parent != m.HomeDir && parent != "/" && parent != "." {
			m.Local.SaveCursor()
			return m, m.loadLocalDir(parent)
		}
		m.Local.Status = "Достигнута домашняя директория"
	case "l", "enter":
		if len(m.Local.Files) == 0 {
			return m, nil
		}
		file := m.Local.Files[m.Local.Cursor]
		m.Local.SaveCursor()
		if file.IsDir {
			return m, m.loadLocalDir(filepath.Join(m.Local.Cwd, file.Name))
		}
		localPath := filepath.Join(m.Local.Cwd, file.Name)
		remotePath := filepath.Join(m.Remote.Cwd, file.Name)
		return m, func() tea.Msg {
			err := sftpclient.UploadFile(m.SftpClient, localPath, remotePath)
			if err != nil {
				return err
			}
			return fmt.Sprintf("Файл %s загружен на сервер!", file.Name)
		}
	}
	return m, nil
}

func (m *Model) handleRemoteKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.Remote.Initialized {
		m.Remote.Initialized = true
		return m, m.loadRemoteDir(".")
	}

	switch key.String() {
	case "k", "up":
		m.Remote.MoveUp()
	case "j", "down":
		m.Remote.MoveDown()
	case "h", "backspace":
		if m.Remote.Cwd != "." && m.Remote.Cwd != "/" {
			m.Remote.SaveCursor()
			parent := filepath.Dir(m.Remote.Cwd)
			return m, m.loadRemoteDir(parent)
		}
	case "l", "enter":
		if len(m.Remote.Files) == 0 {
			return m, nil
		}
		file := m.Remote.Files[m.Remote.Cursor]
		m.Remote.SaveCursor()
		if file.IsDir {
			return m, m.loadRemoteDir(filepath.Join(m.Remote.Cwd, file.Name))
		}
		filePath := filepath.Join(m.Remote.Cwd, file.Name)
		return m, func() tea.Msg {
			localPath := filepath.Join(m.Local.Cwd, file.Name)
			err := sftpclient.DownloadFile(m.SftpClient, filePath, localPath)
			if err != nil {
				return err
			}
			return fmt.Sprintf("Файл %s скачан!", file.Name)
		}
	}
	return m, nil
}

func (m *Model) handleConnListKey(key tea.KeyMsg) {
	switch key.String() {
	case "j", "down":
		if m.ConnCursor < len(m.Connections)-1 {
			m.ConnCursor++
		}
	case "k", "up":
		if m.ConnCursor > 0 {
			m.ConnCursor--
		}
	case "c":
		m.ConnForm.Editing = true
		m.ConnForm.Focused = 0
		m.ConnForm.Inputs = [3]string{"", "", ""}
		m.ConnSubFocus = "form"
	case "l", "enter":
		m.connectToSelected()
	}
}

func (m *Model) connectToSelected() {
	if len(m.Connections) == 0 {
		m.ConnStatus = "Нет сохраненных соединений"
		return
	}
	conn := m.Connections[m.ConnCursor]

	if m.SftpClient != nil {
		m.SftpClient.Close()
		m.SftpClient = nil
	}
	if m.SshClient != nil {
		m.SshClient.Close()
		m.SshClient = nil
	}

	sshClient, err := sftpclient.ConnectSSHWithKeyPath(conn.Name, conn.Host, conn.Path)
	if err != nil {
		m.ConnStatus = "Ошибка подключения: " + err.Error()
		return
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		m.ConnStatus = "Ошибка создания SFTP клиента: " + err.Error()
		return
	}

	m.SshClient = sshClient
	m.SftpClient = sftpClient
	m.Remote = model.NewFileManager()
	m.Focus = "remote"
	m.ConnStatus = "Подключение успешно!"
}
