package database

import "time"

// Message matches the existing message table layout.
type Message struct {
	ID          int64
	CreatedBy   string
	CreatedTime time.Time
	UpdatedBy   string
	UpdatedTime time.Time
	Version     int
	Deleted     int64
	Content     string
}

// BuildMessageContent returns the text persisted into the content column.
func BuildMessageContent(title, subtitle, body string) string {
	switch {
	case body != "":
		return body
	case title != "" && subtitle != "":
		return title + "\n" + subtitle
	case title != "":
		return title
	case subtitle != "":
		return subtitle
	default:
		return "Empty Message"
	}
}
