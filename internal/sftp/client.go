package sftpclient

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"go.dev-cli.fm/internal/model"
	"golang.org/x/crypto/ssh"
)

// ConnectSSHFromJSON парсит JSON, выбирает первую запись и пытается подключиться по SSH
// Подключение пробует в 3 этапа: без аутентификации, с ключом, с паролем
func ConnectSSH(user, addr, keyPath, password string) (*ssh.Client, error) {
	conn := model.Connection{
		Name:     user,
		Host:     addr,
		Path:     keyPath,
		Password: password,
	}

	address := conn.Host
	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:22", address)
	}

	configs := []*ssh.ClientConfig{
		{
			User:            conn.Name,
			Auth:            []ssh.AuthMethod{},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		},
	}

	if conn.Path != "" {
		key, err := os.ReadFile(conn.Path)
		if err == nil {
			if signer, err := ssh.ParsePrivateKey(key); err == nil {
				configs = append(configs, &ssh.ClientConfig{
					User:            conn.Name,
					Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
					Timeout:         10 * time.Second,
				})
			}
		}
	}

	if conn.Password != "" {
		configs = append(configs, &ssh.ClientConfig{
			User:            conn.Name,
			Auth:            []ssh.AuthMethod{ssh.Password(conn.Password)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		})
	}

	var lastErr error
	for _, cfg := range configs {
		client, err := ssh.Dial("tcp", address, cfg)
		if err == nil {
			return client, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("ssh connection failed: %w", lastErr)
	}
	return nil, fmt.Errorf("ssh connection failed with unknown error")
}

func DownloadFile(client *sftp.Client, remotePath, localPath string) error {
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

func UploadFile(client *sftp.Client, localPath, remotePath string) error {
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
