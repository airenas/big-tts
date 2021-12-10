package joiner

import (
	"context"
	"errors"
	"testing"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/stretchr/testify/assert"
)

func TestNewWorker(t *testing.T) {
	got, err := NewWorker("{}.txt", "new{}.txt", "save{}", nil)
	assert.Nil(t, err)
	assert.NotNil(t, got)
	_, err = NewWorker(".txt", "new{}.txt", "Save{}", nil)
	assert.NotNil(t, err)
	_, err = NewWorker("{}.txt", "new.txt", "Save{}", nil)
	assert.NotNil(t, err)
	_, err = NewWorker("{}.txt", "new{}.txt", "Save", nil)
	assert.NotNil(t, err)
}

func TestWorker_Do(t *testing.T) {
	got, err := NewWorker("in/{}", "new/{}/", "save/{}/", []string{"aaa=bbb", "ccc=ddd"})
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		if files == 0 {
			assert.Equal(t, "in/id1/0000.mp3", s)
		}
		files++
		return files < 3
	}
	testCreate := "new/id1/"
	got.createDirFunc = func(s string) error {
		assert.Equal(t, testCreate, s)
		testCreate = "save/id1/"
		return nil
	}
	got.saveFunc = func(s string, b []byte) error {
		assert.Equal(t, "save/id1/list.txt", s)
		assert.Equal(t, "file 'in/id1/0000.mp3'\nfile 'in/id1/0001.mp3'\n", string(b))
		return nil
	}
	got.convertFunc = func(s []string) error {
		assert.Equal(t, []string{"ffmpeg", "-f", "concat",
			"-safe", "0",
			"-i", "save/id1/list.txt",
			"-c", "copy",
			"-metadata", "aaa=bbb",
			"-metadata", "ccc=ddd",
			"new/id1/result.mp3"}, s)
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.Nil(t, err)
}

func TestWorker_Do_Fail(t *testing.T) {
	got, err := NewWorker("in/{}", "new/{}/", "save/{}/", nil)
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		files++
		return files < 3
	}
	got.createDirFunc = func(s string) error {
		return errors.New("err")
	}
	got.saveFunc = func(s string, b []byte) error {
		return nil
	}
	got.convertFunc = func(s []string) error {
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.NotNil(t, err)
}

func TestWorker_Do_FailSave(t *testing.T) {
	got, err := NewWorker("in/{}", "new/{}/", "save/{}/", nil)
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		files++
		return files < 3
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	got.saveFunc = func(s string, b []byte) error {
		return errors.New("err")
	}
	got.convertFunc = func(s []string) error {
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.NotNil(t, err)
}

func TestWorker_Do_FailConvert(t *testing.T) {
	got, err := NewWorker("in/{}", "new/{}/", "save/{}/", nil)
	assert.Nil(t, err)
	files := 0
	got.existsFunc = func(s string) bool {
		files++
		return files < 3
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	got.saveFunc = func(s string, b []byte) error {
		return nil
	}
	got.convertFunc = func(s []string) error {
		return errors.New("err")
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}, OutputFormat: "mp3"})
	assert.NotNil(t, err)
}

func Test_runCmd(t *testing.T) {
	type args struct {
		cmdArr []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "OK", args: args{cmdArr: []string{"echo", "1"}}, wantErr: false},
		{name: "OK", args: args{cmdArr: []string{"missingExec123", "1"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runCmd(tt.args.cmdArr); (err != nil) != tt.wantErr {
				t.Errorf("runCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
