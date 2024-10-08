package synthesizer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/pkg/errors"
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

func TestWorker_Do_Exit_OnCancel(t *testing.T) {
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
	ctx, cFunc := context.WithCancel(context.Background())
	cFunc()
	err = got.Do(ctx, &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.Equal(t, context.Canceled, err)
}

func TestWorker_Do_Exit_OnFailure(t *testing.T) {
	got, err := NewWorker("in/{}", "new/{}/", "url", 10)
	assert.Nil(t, err)
	got.existsFunc = func(s string) bool {
		return strings.HasSuffix(s, ".txt")
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	got.loadFunc = func(s string) ([]byte, error) {
		return []byte("in"), nil
	}
	var tCnt int64
	tErr := errors.New("err")
	got.callFunc = func(s string, tm *messages.TTSMessage) ([]byte, error) {
		if atomic.AddInt64(&tCnt, 1) == 1 {
			time.Sleep(time.Millisecond * 10)
		} else {
			time.Sleep(time.Millisecond * 100)
		}
		return nil, tErr
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.Equal(t, tErr, err)
	assert.Equal(t, int64(10), tCnt)
}

func TestWorker_Do_WithRealInvoke(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		res := result{}
		res.AudioAsString = base64.StdEncoding.EncodeToString([]byte("audio data"))
		_ = json.NewEncoder(rw).Encode(res)
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

// func TestWorker_Do_WithRealInvokeFail(t *testing.T) {
// 	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
// 		rw.WriteHeader(http.StatusBadRequest)
// 	}))
// 	defer srv.Close()

// 	got, err := NewWorker("in/{}", "new/{}/", srv.URL, 1)
// 	require.Nil(t, err)
// 	files := 0
// 	got.existsFunc = func(s string) bool {
// 		files++
// 		return files < 2
// 	}
// 	got.createDirFunc = func(s string) error {
// 		return nil
// 	}
// 	got.loadFunc = func(s string) ([]byte, error) {
// 		return []byte("in"), nil
// 	}
// 	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
// 	assert.NotNil(t, err)
// 	var errTest *utils.ErrNonRestorableUsage
// 	assert.ErrorAs(t, err, &errTest)
// }

func Test_isNonRestorableErrCode(t *testing.T) {
	tests := []struct {
		name string
		args int
		want bool
	}{
		{name: "400", args: 400, want: false},
		{name: "409", args: 409, want: false},
		{name: "500", args: 500, want: false},
		{name: "300", args: 300, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNonRestorableErrCode(tt.args); got != tt.want {
				t.Errorf("isNonRestorableErrCose() = %v, want %v", got, tt.want)
			}
		})
	}
}
