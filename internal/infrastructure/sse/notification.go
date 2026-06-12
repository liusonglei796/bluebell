package sse

type Notification struct {
	Payload interface{} `json:"payload"`
	Type    string      `json:"type"`
}
