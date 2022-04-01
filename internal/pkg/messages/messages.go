package messages

import (
	amessages "github.com/airenas/async-api/pkg/messages"
)

const (
	st = "BigTTS/"
	// Upload queue name
	Upload = st + "Upload"
	// Split queue name
	Split = st + "Split"
	// Synthesize queue name
	Synthesize = st + "Synthesize"
	// Join queue name
	Join = st + "Join"
	// Fail queue name
	Fail = st + "Fail"
	// Inform  queue name
	Inform = st + "Inform"
)

// TTSMessage main message passing through in big tts system
type TTSMessage struct {
	amessages.QueueMessage
	Voice        string   `json:"voice,omitempty"`
	SaveRequest  bool     `json:"saveRequest,omitempty"`
	Speed        float64  `json:"speed,omitempty"`
	OutputFormat string   `json:"outputFormat,omitempty"`
	SaveTags     []string `json:"tags,omitempty"`
	RequestID    string   `json:"requestID,omitempty"`
}

// NewMessageFrom creates a copy of a message
func NewMessageFrom(m *TTSMessage) *TTSMessage {
	return &TTSMessage{QueueMessage: m.QueueMessage, Voice: m.Voice, SaveRequest: m.SaveRequest,
		Speed: m.Speed, SaveTags: m.SaveTags, OutputFormat: m.OutputFormat, RequestID: m.RequestID}
}
