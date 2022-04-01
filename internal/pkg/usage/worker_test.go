package usage

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/stretchr/testify/assert"
)

func TestNewWorker(t *testing.T) {
	got, err := NewWorker("url")
	assert.Nil(t, err)
	assert.NotNil(t, got)
	_, err = NewWorker("")
	assert.NotNil(t, err)
}

func TestWorker_Do(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/tt/restore/m:rid", r.RequestURI)
		b, _ := ioutil.ReadAll(r.Body)
		assert.Equal(t, `{"error":"err"}`, string(b))
		rw.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	got, err := NewWorker(srv.URL)
	assert.Nil(t, err)
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1", Error: "err"},
		RequestID: "tt:m:rid"})
	assert.Nil(t, err)
}
