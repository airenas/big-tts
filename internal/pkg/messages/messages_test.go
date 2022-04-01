package messages

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMessageFrom(t *testing.T) {
	assert.Equal(t, &TTSMessage{SaveRequest: true, RequestID: "rID", Voice: "astra"},
		NewMessageFrom(&TTSMessage{SaveRequest: true, RequestID: "rID", Voice: "astra"}))
}
