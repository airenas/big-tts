package synthesizer

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"encoding/json"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/stretchr/testify/assert"
)

func TestNewWorker(t *testing.T) {
	got, err := NewWorker("{}.txt", "new{}.txt", "url", 1)
	assert.Nil(t, err)
	assert.NotNil(t, got)
	_, err = NewWorker(".txt", "new{}.txt", "url", 1)
	assert.NotNil(t, err)
	_, err = NewWorker("{}.txt", "new.txt", "url", 1)
	assert.NotNil(t, err)
	_, err = NewWorker("{}.txt", "new{}.txt", "", 1)
	assert.NotNil(t, err)
	_, err = NewWorker("{}.txt", "new{}.txt", "url", 0)
	assert.NotNil(t, err)
}

func TestWorker_Do(t *testing.T) {
	got, err := NewWorker("in/{}", "new/{}/", "url", 1)
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		files++
		if files == 1 {
			assert.Equal(t, "in/id1/0000.txt", s)
			return true
		}
		if files == 2 {
			assert.Equal(t, "new/id1/0000.mp3", s)
			return false
		}
		return false
	}
	got.createDirFunc = func(s string) error {
		assert.Equal(t, "new/id1/", s)
		return nil
	}
	got.loadFunc = func(s string) ([]byte, error) {
		assert.Equal(t, "in/id1/0000.txt", s)
		return []byte("olia"), nil
	}
	got.callFunc = func(s string, tm *messages.TTSMessage) ([]byte, error) {
		assert.Equal(t, "olia", s)
		return []byte("done"), nil
	}
	got.saveFunc = func(s string, b []byte) error {
		assert.Equal(t, "new/id1/0000.mp3", s)
		assert.Equal(t, "done", string(b))
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.Nil(t, err)
}

func TestWorker_Do_Exists_Skip(t *testing.T) {
	got, err := NewWorker("in/{}", "new/{}/", "url", 1)
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		files++
		if files == 1 {
			assert.Equal(t, "in/id1/0000.txt", s)
			return true
		}
		if files == 2 {
			assert.Equal(t, "new/id1/0000.mp3", s)
			return true
		}
		return false
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	got.loadFunc = func(s string) ([]byte, error) {
		t.Error("not expected")
		return nil, nil
	}
	got.callFunc = func(s string, tm *messages.TTSMessage) ([]byte, error) {
		t.Error("not expected")
		return nil, nil
	}
	got.saveFunc = func(s string, b []byte) error {
		t.Error("not expected")
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.Nil(t, err)
}

func TestWorker_Do_Fail_Invoke(t *testing.T) {
	got, err := NewWorker("in/{}", "new/{}/", "url", 1)
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		files++
		return files < 2
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	got.loadFunc = func(s string) ([]byte, error) {
		return nil, errors.New("err")
	}
	got.callFunc = func(s string, tm *messages.TTSMessage) ([]byte, error) {
		t.Error("not expected")
		return nil, nil
	}
	got.saveFunc = func(s string, b []byte) error {
		t.Error("not expected")
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.NotNil(t, err)
}

func TestWorker_Do_Fail_Calc(t *testing.T) {
	got, err := NewWorker("in/{}", "new/{}/", "url", 1)
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		files++
		return files < 2
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	got.loadFunc = func(s string) ([]byte, error) {
		return []byte("in"), nil
	}
	got.callFunc = func(s string, tm *messages.TTSMessage) ([]byte, error) {
		return nil, errors.New("err")
	}
	got.saveFunc = func(s string, b []byte) error {
		t.Error("not expected")
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.NotNil(t, err)
}

func TestWorker_Do_WithRealInvoke(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		res := result{}
		res.AudioAsString = base64.StdEncoding.EncodeToString([]byte("audio data"))
		json.NewEncoder(rw).Encode(res)
	}))
	defer srv.Close()

	got, err := NewWorker("in/{}", "new/{}/", srv.URL, 1)
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		files++
		return files < 2
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	got.loadFunc = func(s string) ([]byte, error) {
		return []byte("in"), nil
	}
	got.saveFunc = func(s string, b []byte) error {
		assert.Equal(t, "audio data", string(b))
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.Nil(t, err)
}

func TestWorker_Do_WithRealInvokeFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	got, err := NewWorker("in/{}", "new/{}/", srv.URL, 1)
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		files++
		return files < 2
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	got.loadFunc = func(s string) ([]byte, error) {
		return []byte("in"), nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.NotNil(t, err)
}
