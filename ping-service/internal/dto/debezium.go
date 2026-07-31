package dto

import "encoding/json"

type DebeziumMessage struct {
	ID        string `json:"-"`
	TopicName string `json:"-"`

	Before    json.RawMessage `json:"before"`
	After     json.RawMessage `json:"after"`
	Operation string          `json:"op"` // c=create, r=read, u=update, d=delete
	Source    DataSource      `json:"source"`
}

type DataSource struct {
	Table string `json:"table"`
}
