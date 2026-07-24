package database

import "time"

// Message matches the existing message table layout.
type Message struct {
	ID          int64
	DeviceKey   string
	Category    string
	Title       string
	Body        string
	PushParams  map[string]interface{}
	CreatedBy   string
	CreatedTime time.Time
	UpdatedBy   string
	UpdatedTime time.Time
	Version     int
	Deleted     int64
}
