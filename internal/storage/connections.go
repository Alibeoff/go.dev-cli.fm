package storage

import (
	"encoding/json"
	"os"

	"go.dev-cli.fm/internal/model"
)

func LoadConnections(filename string) ([]model.Connection, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var conns []model.Connection
	err = json.Unmarshal(data, &conns)
	return conns, err
}

func SaveConnections(filename string, conns []model.Connection) error {
	data, err := json.MarshalIndent(conns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}
