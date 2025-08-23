package model

type Connection struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Path     string `json:"path"`
}
