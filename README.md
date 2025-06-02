# Terminal SFTP File Manager

A terminal-based file manager written in Go with support for local and remote (SFTP) file browsing and file transfers.  
This project uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the terminal UI and [go-sftp](https://github.com/pkg/sftp) for SFTP operations.

---

## Features

- Browse local filesystem.
- Manage saved SSH connections.
- Connect to remote servers via SFTP using SSH keys.
- Upload and download files between local and remote.
- Delete files locally and remotely.
- Intuitive keyboard navigation and commands.
- Three panes: Local, Connections, and Remote (SFTP) file managers.
- Remote pane appears only after connecting to a selected server.

---

## Installation

### Prerequisites

- Go 1.18+ installed ([download](https://golang.org/dl/))
- SSH private key for authentication (usually `~/.ssh/id_rsa`)

### Get the code

```
git clone https://github.com/yourusername/terminal-sftp-fm.git
cd terminal-sftp-fm
```

### Install dependencies

```
go mod tidy
```

### Build and run

```
go build -o sftp-fm
./sftp-fm
```

---

## Usage

- On start, you see two panes:  
  - **Local File Manager**  
  - **Connections Manager** (list of saved SSH connections)

- Use `Tab` to switch focus between panes.

- In **Connections Manager**:  
  - `j`/`k` or arrow keys to move selection.  
  - `c` to add a new connection (fill form fields).  
  - In form:  
    - Enter to move between fields.  
    - `Ctrl+A` to save connection.  
    - `Ctrl+C` to cancel.  
  - Select a connection and press `l` or `Enter` to connect via SFTP.  
  - `d` deletes selected connection.

- After connecting, the **Remote File Manager** pane appears.  
  - Navigate remote files similarly to local.  
  - Use `l` or `Enter` to enter directories or download files.  
  - Use `h` or `Backspace` to go up one directory.  
  - `d` deletes remote files.

- In **Local File Manager**:  
  - Navigate files with `j`/`k` or arrows.  
  - Use `l` or `Enter` to enter directories or upload files to remote.  
  - Use `h` or `Backspace` to go up one directory.  
  - `d` deletes local files.

- Quit program with `q` or `Ctrl+C`.

---

## Keybindings

| Key           | Description                                      |
|---------------|--------------------------------------------------|
| `Tab`         | Switch focus between panes                       |
| `j` / `Down`  | Move cursor down                                 |
| `k` / `Up`    | Move cursor up                                   |
| `l` / `Enter` | Enter directory / open file / connect to server  |
| `h` / `Backspace` | Go up to parent directory                    |
| `c`           | Add new connection (in Connections pane)         |
| `Ctrl+A`      | Save new connection (in form)                    |
| `Ctrl+C`      | Cancel form or quit program                      |
| `d`           | Delete selected file or connection               |
| `q`           | Quit program                                     |
---

## Configuration

- Connections are saved in `connections.json` in the current directory.  
- Format example:

```
[
  {
    "name": "myserver",
    "host": "example.com:22",
    "path": "/home/user/.ssh/id_rsa"
  }
]
```

---

## Dependencies

- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) — Terminal UI framework  
- [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) — Styling for terminal UI  
- [github.com/pkg/sftp](https://github.com/pkg/sftp) — SFTP client library  
- [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh) — SSH client library  

---

## License

MIT License © Virs Core

---

## Contributing

Feel free to open issues or submit pull requests!

---

Thank you for using Terminal SFTP File Manager!

---
