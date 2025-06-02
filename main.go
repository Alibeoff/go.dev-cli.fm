package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	listWidth  = 32
	listHeight = 24
)

var (
	borderColor       = lipgloss.Color("#7D56F4")
	remoteBorderColor = lipgloss.Color("#9B59B6")
	connBorderColor   = borderColor
	cursorColor       = lipgloss.Color("#7D56F4")
	titleStyle        = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(borderColor).
				Padding(0, 1).
				MarginBottom(1)
	cursorStyle = lipgloss.NewStyle().
			Foreground(cursorColor).
			Bold(true)
	dirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00BFFF")).
			Bold(true)
	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			MarginTop(1).
			Italic(true)
)

type fileEntry struct {
	Name  string
	IsDir bool
	Info  os.FileInfo
}

type fileManager struct {
	cwd         string
	files       []fileEntry
	cursor      int
	status      string // статус отображается только во время загрузки или ошибки
	cursorStack map[string]int
	scroll      int
}

type Connection struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Path string `json:"path"`
}

func loadConnections(filename string) ([]Connection, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // файл не существует — пустой список
		}
		return nil, err
	}
	var conns []Connection
	err = json.Unmarshal(data, &conns)
	return conns, err
}

func saveConnections(filename string, conns []Connection) error {
	data, err := json.MarshalIndent(conns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func newFileManager() fileManager {
	return fileManager{
		cwd:         ".",
		files:       []fileEntry{},
		cursor:      0,
		status:      "", // пустой статус по умолчанию
		cursorStack: map[string]int{".": 0},
	}
}

func (fm *fileManager) calcScroll() {
	total := len(fm.files)
	if total <= listHeight {
		fm.scroll = 0
		return
	}
	if fm.cursor < 0 {
		fm.scroll = 0
		return
	}
	if fm.cursor >= total {
		fm.scroll = total - listHeight
		return
	}
	if fm.cursor < listHeight/2 {
		fm.scroll = 0
		return
	}
	if fm.cursor > total-listHeight/2 {
		fm.scroll = total - listHeight
		return
	}
	fm.scroll = fm.cursor - listHeight/2
}

func (fm *fileManager) setFiles(files []fileEntry) {
	fm.files = files
	if pos, ok := fm.cursorStack[fm.cwd]; ok && pos < len(files) {
		fm.cursor = pos
	} else {
		fm.cursor = 0
	}
	fm.calcScroll()
	// не устанавливаем статус с путём, статус будет пустым после загрузки
}

func (fm *fileManager) saveCursor() {
	fm.cursorStack[fm.cwd] = fm.cursor
}

func (fm *fileManager) moveUp() {
	if fm.cursor > 0 {
		fm.cursor--
		fm.calcScroll()
	}
}

func (fm *fileManager) moveDown() {
	if fm.cursor < len(fm.files)-1 {
		fm.cursor++
		fm.calcScroll()
	}
}

func (fm *fileManager) render(focused bool, title string, borderStyle lipgloss.Style, focusColor lipgloss.Color) string {
	var b strings.Builder
	if len(fm.files) == 0 {
		b.WriteString("  (пусто или загрузка...)\n")
		for i := 1; i < listHeight; i++ {
			b.WriteString("\n")
		}
	} else {
		for i := 0; i < listHeight; i++ {
			fileIdx := fm.scroll + i
			if fileIdx >= len(fm.files) {
				b.WriteString("\n")
				continue
			}
			file := fm.files[fileIdx]
			cursor := "  "
			lineStyle := fileStyle
			if fm.cursor == fileIdx && focused {
				cursor = cursorStyle.Render("▶ ")
			}
			icon := "📄"
			if file.IsDir {
				icon = "📁"
				lineStyle = dirStyle
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, icon, lineStyle.Render(file.Name)))
		}
	}
	fileList := borderStyle.Render(b.String())
	titleStyleFocused := titleStyle.Copy()
	if focused {
		titleStyleFocused = titleStyleFocused.Foreground(focusColor)
	} else {
		titleStyleFocused = titleStyleFocused.Foreground(lipgloss.Color("#888888"))
	}
	header := titleStyleFocused.Render(" " + title + " ")
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		fileList,
		// показываем статус только если он не пустой
		lipgloss.NewStyle().Render(fm.status),
	)
}

type connForm struct {
	inputs  [3]string // name, host, path
	focused int       // индекс текущего активного инпута
	editing bool      // флаг, что форма открыта для редактирования
}

type model struct {
	sftpClient *sftp.Client
	sshClient  *ssh.Client

	remote  fileManager
	local   fileManager
	focus   string // "local", "conn", "remote"
	homeDir string

	connections  []Connection
	connForm     connForm
	connCursor   int
	connSubFocus string // "list" или "form" — подфокус внутри окна Connection FM

	lastKeyPress time.Time
	lastKey      string
}

type filesMsg struct {
	cwd   string
	files []os.FileInfo
	err   error
}

type localFilesMsg struct {
	cwd   string
	files []os.FileInfo
	err   error
}

func initialModel() model {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	startDir, err := os.Getwd()
	if err != nil {
		startDir = homeDir
	}
	localFM := newFileManager()
	localFM.cwd = startDir
	localFM.status = ""
	localFM.cursorStack = map[string]int{startDir: 0}

	conns, err := loadConnections("connections.json")
	if err != nil {
		log.Printf("Ошибка загрузки connections.json: %v", err)
		conns = nil
	}

	return model{
		sftpClient:   nil,
		sshClient:    nil,
		local:        localFM,
		remote:       newFileManager(),
		focus:        "local",
		homeDir:      homeDir,
		connections:  conns,
		connForm:     connForm{inputs: [3]string{"", "", ""}, focused: 0, editing: false},
		connCursor:   0,
		connSubFocus: "list",
	}
}

func (m model) loadRemoteDir(path string) tea.Cmd {
	return func() tea.Msg {
		m.remote.status = "Загрузка..."
		files, err := m.sftpClient.ReadDir(path)
		return filesMsg{cwd: path, files: files, err: err}
	}
}

func (m model) loadLocalDir(path string) tea.Cmd {
	return func() tea.Msg {
		m.local.status = "Загрузка..."
		entries, err := os.ReadDir(path)
		if err != nil {
			return localFilesMsg{cwd: path, files: nil, err: err}
		}
		var infos []os.FileInfo
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			infos = append(infos, info)
		}
		return localFilesMsg{cwd: path, files: infos, err: nil}
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.loadLocalDir(m.local.cwd),
	)
}

func (m *model) renderConnections() string {
	var b strings.Builder
	b.WriteString("Сохранённые соединения:\n\n")
	if len(m.connections) == 0 {
		b.WriteString("  (нет сохранённых серверов)\n\n")
	} else {
		for i, c := range m.connections {
			cursor := "  "
			if i == m.connCursor && m.focus == "conn" && m.connSubFocus == "list" {
				cursor = cursorStyle.Render("▶ ")
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, c.Name))
		}
	}
	b.WriteString("\n")

	if m.connForm.editing && m.connSubFocus == "form" {
		b.WriteString("Добавить новое соединение (Ctrl+A - сохранить, Ctrl+C - отмена):\n")
		labels := []string{"Имя: ", "Host: ", "Путь к ключу: "}
		for i, label := range labels {
			line := label + m.connForm.inputs[i]
			if m.connForm.focused == i {
				line = cursorStyle.Render(line + "_")
			}
			b.WriteString(line + "\n")
		}
	} else {
		b.WriteString("Нажмите 'c' для добавления нового соединения\n")
	}
	return b.String()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case filesMsg:
		if msg.err != nil {
			m.remote.status = "Ошибка: " + msg.err.Error()
			m.remote.files = []fileEntry{}
		} else {
			m.remote.cwd = msg.cwd
			var entries []fileEntry
			for _, f := range msg.files {
				entries = append(entries, fileEntry{
					Name:  f.Name(),
					IsDir: f.IsDir(),
					Info:  f,
				})
			}
			m.remote.setFiles(entries)
			m.remote.status = ""
		}
		return m, nil
	case localFilesMsg:
		if msg.err != nil {
			m.local.status = "Ошибка: " + msg.err.Error()
			m.local.files = []fileEntry{}
		} else {
			m.local.cwd = msg.cwd
			var entries []fileEntry
			for _, f := range msg.files {
				entries = append(entries, fileEntry{
					Name:  f.Name(),
					IsDir: f.IsDir(),
					Info:  f,
				})
			}
			m.local.setFiles(entries)
			m.local.status = ""
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			switch m.focus {
			case "local":
				m.focus = "conn"
			case "conn":
				if m.sftpClient != nil {
					m.focus = "remote"
				} else {
					m.focus = "local"
				}
			case "remote":
				m.focus = "local"
			}
			return m, nil
		}
		// Удаление элементов по 'd' только если подфокус "list"
		if msg.String() == "d" && m.focus == "conn" && m.connSubFocus == "list" {
			if len(m.connections) > 0 {
				m.connections = append(m.connections[:m.connCursor], m.connections[m.connCursor+1:]...)
				if m.connCursor >= len(m.connections) && m.connCursor > 0 {
					m.connCursor--
				}
				_ = saveConnections("connections.json", m.connections)
			}
			return m, nil
		}
		// Удаление файлов в локальном и удалённом менеджерах (если не в форме)
		if msg.String() == "d" && m.connSubFocus != "form" {
			if m.focus == "local" && len(m.local.files) > 0 {
				selected := m.local.files[m.local.cursor]
				path := filepath.Join(m.local.cwd, selected.Name)
				err := os.RemoveAll(path)
				if err != nil {
					m.local.status = "Ошибка удаления: " + err.Error()
				} else {
					m.local.status = "Файл удалён: " + selected.Name
					return m, m.loadLocalDir(m.local.cwd)
				}
			} else if m.focus == "remote" && len(m.remote.files) > 0 {
				selected := m.remote.files[m.remote.cursor]
				path := filepath.Join(m.remote.cwd, selected.Name)
				err := m.sftpClient.Remove(path)
				if err != nil {
					m.remote.status = "Ошибка удаления: " + err.Error()
				} else {
					m.remote.status = "Файл удалён: " + selected.Name
					return m, m.loadRemoteDir(m.remote.cwd)
				}
			}
			return m, nil
		}

		switch m.focus {
		case "local":
			switch msg.String() {
			case "k", "up":
				m.local.moveUp()
			case "j", "down":
				m.local.moveDown()
			case "h", "backspace":
				if m.local.cwd != m.homeDir && m.local.cwd != "/" && m.local.cwd != "." {
					m.local.saveCursor()
					parent := filepath.Dir(m.local.cwd)
					rel, err := filepath.Rel(m.homeDir, parent)
					if err == nil && (rel == "." || !strings.HasPrefix(rel, "..")) {
						return m, m.loadLocalDir(parent)
					}
					m.local.status = "Достигнута домашняя директория"
				}
			case "l", "enter":
				if len(m.local.files) == 0 {
					break
				}
				selected := m.local.files[m.local.cursor]
				m.local.saveCursor()
				if selected.IsDir {
					return m, m.loadLocalDir(filepath.Join(m.local.cwd, selected.Name))
				} else {
					localPath := filepath.Join(m.local.cwd, selected.Name)
					remotePath := filepath.Join(m.remote.cwd, selected.Name)
					return m, func() tea.Msg {
						err := uploadFile(m.sftpClient, localPath, remotePath)
						if err != nil {
							return err
						}
						return fmt.Sprintf("Файл %s загружен на сервер!", selected.Name)
					}
				}
			}
		case "remote":
			switch msg.String() {
			case "k", "up":
				m.remote.moveUp()
			case "j", "down":
				m.remote.moveDown()
			case "h", "backspace":
				if m.remote.cwd != "." && m.remote.cwd != "/" {
					m.remote.saveCursor()
					parent := filepath.Dir(m.remote.cwd)
					return m, m.loadRemoteDir(parent)
				}
			case "l", "enter":
				if len(m.remote.files) == 0 {
					break
				}
				selected := m.remote.files[m.remote.cursor]
				m.remote.saveCursor()
				if selected.IsDir {
					return m, m.loadRemoteDir(filepath.Join(m.remote.cwd, selected.Name))
				} else {
					filePath := filepath.Join(m.remote.cwd, selected.Name)
					return m, func() tea.Msg {
						localPath := filepath.Join(m.local.cwd, selected.Name)
						err := downloadFile(m.sftpClient, filePath, localPath)
						if err != nil {
							return err
						}
						return fmt.Sprintf("Файл %s скачан!", selected.Name)
					}
				}
			}
		case "conn":
			if m.connForm.editing && m.connSubFocus == "form" {
				switch msg.String() {
				case "esc", "ctrl+c":
					m.connForm = connForm{inputs: [3]string{"", "", ""}, focused: 0, editing: false}
					m.connSubFocus = "list"
				case "ctrl+a":
					newConn := Connection{
						Name: m.connForm.inputs[0],
						Host: m.connForm.inputs[1],
						Path: m.connForm.inputs[2],
					}
					if newConn.Name != "" && newConn.Host != "" && newConn.Path != "" {
						m.connections = append(m.connections, newConn)
						_ = saveConnections("connections.json", m.connections)
					}
					m.connForm = connForm{inputs: [3]string{"", "", ""}, focused: 0, editing: false}
					m.connCursor = len(m.connections) - 1
					m.connSubFocus = "list"
				case "enter":
					m.connForm.focused = (m.connForm.focused + 1) % 3
				default:
					if len(msg.Runes) > 0 {
						r := msg.Runes[0]
						m.connForm.inputs[m.connForm.focused] += string(r)
					} else if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
						input := m.connForm.inputs[m.connForm.focused]
						if len(input) > 0 {
							m.connForm.inputs[m.connForm.focused] = input[:len(input)-1]
						}
					}
				}
			} else if m.connSubFocus == "list" {
				switch msg.String() {
				case "j", "down":
					if len(m.connections) > 0 && m.connCursor < len(m.connections)-1 {
						m.connCursor++
					}
				case "k", "up":
					if m.connCursor > 0 {
						m.connCursor--
					}
				case "c":
					m.connForm.editing = true
					m.connForm.focused = 0
					m.connForm.inputs = [3]string{"", "", ""}
					m.connSubFocus = "form"
				case "l", "enter":
					if len(m.connections) > 0 {
						conn := m.connections[m.connCursor]

						// Закрываем старое соединение
						if m.sftpClient != nil {
							m.sftpClient.Close()
							m.sftpClient = nil
						}
						if m.sshClient != nil {
							m.sshClient.Close()
							m.sshClient = nil
						}

						sshClient, err := connectSSHWithKeyPath(conn.Name, conn.Host, conn.Path)
						if err != nil {
							m.local.status = "Ошибка подключения: " + err.Error()
							return m, nil
						}

						sftpClient, err := sftp.NewClient(sshClient)
						if err != nil {
							sshClient.Close()
							m.local.status = "Ошибка создания SFTP клиента: " + err.Error()
							return m, nil
						}

						m.sshClient = sshClient
						m.sftpClient = sftpClient
						m.remote = newFileManager()
						m.focus = "remote"

						return m, m.loadRemoteDir(".")
					}
				}
			}
		}
		return m, nil
	case string:
		if m.focus == "local" {
			m.local.status = msg
		} else if m.focus == "remote" {
			m.remote.status = msg
		}
	case error:
		if m.focus == "local" {
			m.local.status = "Ошибка: " + msg.Error()
		} else if m.focus == "remote" {
			m.remote.status = "Ошибка: " + msg.Error()
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	localBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(listWidth).
		Height(listHeight)

	connBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(connBorderColor).
		Padding(0, 1).
		Width(listWidth).
		Height(listHeight)

	localView := m.local.render(m.focus == "local", "Локальный ФМ", localBorderStyle, borderColor)

	connTitleColor := lipgloss.Color("#888888")
	if m.focus == "conn" {
		connTitleColor = connBorderColor
	}
	connTitle := titleStyle.Copy().Foreground(connTitleColor).Render(" Connection ФМ ")
	connContent := m.renderConnections()
	connView := lipgloss.JoinVertical(lipgloss.Left,
		connTitle,
		connBorderStyle.Render(connContent),
		statusStyle.Render(""),
	)

	if m.sftpClient == nil {
		content := lipgloss.JoinHorizontal(lipgloss.Top,
			localView,
			lipgloss.NewStyle().Width(4).Render(""),
			connView,
		)
		return content
	}

	remoteBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(remoteBorderColor).
		Padding(0, 1).
		Width(listWidth).
		Height(listHeight)

	remoteView := m.remote.render(m.focus == "remote", "SFTP ФМ", remoteBorderStyle, remoteBorderColor)

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		localView,
		lipgloss.NewStyle().Width(4).Render(""),
		remoteView,
		lipgloss.NewStyle().Width(4).Render(""),
		connView,
	)

	return content
}

func downloadFile(client *sftp.Client, remotePath, localPath string) error {
	remoteFile, err := client.Open(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()
	localFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()
	_, err = io.Copy(localFile, remoteFile)
	return err
}

func uploadFile(client *sftp.Client, localPath, remotePath string) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()
	remoteFile, err := client.Create(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()
	_, err = io.Copy(remoteFile, localFile)
	return err
}

func connectSSHWithKeyPath(user, addr, keyPath string) (*ssh.Client, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	return ssh.Dial("tcp", addr, config)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		log.Fatalf("Ошибка запуска программы: %v", err)
	}
}
