package messages

import (
	amessages "github.com/airenas/async-api/pkg/messages"
)

const (
	st         = "BigTTS/"
	Upload     = st + "Upload"
	Split      = st + "Split"
	Synthesize = st + "Synthesize"
	Join       = st + "Join"

	Inform = st + "Inform"
)

type TTSMessage struct {
	amessages.QueueMessage
	Voice        string   `json:"voice,omitempty"`
	SaveRequest  bool     `json:"saveRequest,omitempty"`
	Speed        float64  `json:"speed,omitempty"`
	OutputFormat string   `json:"outputFormat,omitempty"`
	SaveTags     []string `json:"tags,omitempty"`
}

func NewMessageFrom(m *TTSMessage) *TTSMessage {
	return &TTSMessage{QueueMessage: m.QueueMessage, Voice: m.Voice, SaveRequest: m.SaveRequest,
		Speed: m.Speed, SaveTags: m.SaveTags, OutputFormat: m.OutputFormat}
}
