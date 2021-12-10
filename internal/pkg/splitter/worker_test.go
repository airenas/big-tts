package splitter

import (
	"context"
	"testing"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_getNewPattern(t *testing.T) {
	type args struct {
		str string
		r   rune
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "space", args: args{str: "   ", r: ' '}, want: "   "},
		{name: "dot", args: args{str: "   ", r: '.'}, want: "  ."},
		{name: "newline", args: args{str: "   ", r: '\n'}, want: "  \n"},
		{name: "lower", args: args{str: "   ", r: 'a'}, want: "  -"},
		{name: "upper", args: args{str: "   ", r: 'A'}, want: "  U"},
		{name: "space", args: args{str: "  .", r: ' '}, want: " . "},
		{name: "space", args: args{str: " . ", r: ' '}, want: " . "},
		{name: "newline", args: args{str: " . ", r: '\n'}, want: " .\n"},
		{name: "newline", args: args{str: " .\n", r: '\n'}, want: ".\n\n"},
		{name: "space", args: args{str: " .\n", r: ' '}, want: " .\n"},
		{name: "tab", args: args{str: " .\n", r: '\t'}, want: " .\n"},
		{name: "tab", args: args{str: " ..", r: '\t'}, want: ".. "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNewPattern(tt.args.str, tt.args.r); got != tt.want {
				t.Errorf("getNewPattern() = '%v', want '%v'", got, tt.want)
			}
		})
	}
}

func Test_getNextSplit(t *testing.T) {
	type args struct {
		str      string
		start    int
		interval int
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{name: "len", args: args{str: "aaa aaa aaa aaa aaa", start: 10, interval: 10}, want: 19, wantErr: false},
		{name: "space", args: args{str: "aaa aaa aaa aaa aaa aaa", start: 10, interval: 10}, want: 11, wantErr: false},
		{name: "space", args: args{str: "aaa aaa aaa aaa. Aaa aaa", start: 10, interval: 10}, want: 16, wantErr: false},
		{name: "space", args: args{str: "aaa aaa. Aaa\n\naaa.\n Aaa aaa aaa", start: 5, interval: 20}, want: 19, wantErr: false},
		{name: "space", args: args{str: "aaa aaaaaaaaaaa. aa. aaa", start: 5, interval: 2}, want: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getNextSplit([]rune(tt.args.str), tt.args.start, tt.args.interval)
			if (err != nil) != tt.wantErr {
				t.Errorf("getNextSplit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getNextSplit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewWorker(t *testing.T) {
	got, err := NewWorker("{}.txt", "new{}.txt")
	assert.Nil(t, err)
	assert.NotNil(t, got)
	_, err = NewWorker("{}.txt", "new.txt")
	assert.NotNil(t, err)
	_, err = NewWorker("aaa.txt", "new{}.txt")
	assert.NotNil(t, err)
}

func TestWorker_Do(t *testing.T) {
	got, err := NewWorker("{}.txt", "new/{}/path")
	assert.Nil(t, err)
	got.loadFunc = func(s string) ([]byte, error) {
		assert.Equal(t, "id1.txt", s)
		return []byte("string"), nil
	}
	got.saveFunc = func(s string, b []byte) error {
		return nil
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}})
	assert.Nil(t, err)
}

func TestWorker_Do_Fail(t *testing.T) {
	got, err := NewWorker("{}.txt", "new/{}/path")
	assert.Nil(t, err)
	got.loadFunc = func(s string) ([]byte, error) {
		return nil, errors.New("err")
	}
	got.saveFunc = func(s string, b []byte) error {
		return nil
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}})
	assert.NotNil(t, err)
}

func TestWorker_Do_FailSave(t *testing.T) {
	got, err := NewWorker("{}.txt", "new/{}/path")
	assert.Nil(t, err)
	got.saveFunc = func(s string, b []byte) error {
		return errors.New("err")
	}
	got.createDirFunc = func(s string) error {
		return nil
	}
	err = got.Do(context.Background(), &messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "id1"}})
	assert.NotNil(t, err)
}
