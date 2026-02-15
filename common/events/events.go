package events

// ClickEvent represents a URL redirect event published to the message queue.
// Phase 7: Type definition only. Phase 9 will wire Kafka producer/consumer.
type ClickEvent struct {
	ShortCode string `json:"short_code"`
	Timestamp int64  `json:"timestamp"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
}
