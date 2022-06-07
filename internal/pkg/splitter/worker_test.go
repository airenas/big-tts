package splitter

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/tts-line/pkg/ssml"
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

func Test_toRateStr(t *testing.T) {
	tests := []struct {
		args float32
		want string
	}{
		{args: 2, want: "50%"},
		{args: 1.5, want: "75%"},
		{args: 3, want: "50%"},
		{args: 1, want: "100%"},
		{args: 0.9, want: "120%"},
		{args: 0.5, want: "200%"},
		{args: 0.75, want: "150%"},
		{args: -1.75, want: "200%"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%.2f", tt.args), func(t *testing.T) {
			if got := toRateStr(tt.args); got != tt.want {
				t.Errorf("toRateStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saveToSSMLString(t *testing.T) {
	tests := []struct {
		name string
		args []ssml.Part
		want string
	}{
		{name: "Empty", args: nil, want: "<speak></speak>"},
		{name: "Text", args: []ssml.Part{&ssml.Text{Texts: []ssml.TextPart{{Text: "olia"}}, Speed: 1, Voice: "as"}},
			want: `<speak><voice name="as"><prosody rate="100%">olia</prosody></voice></speak>`},
		{name: "Break", args: []ssml.Part{&ssml.Pause{Duration: 4 * time.Second}},
			want: `<speak><break time="4000ms"/></speak>`},
		{name: "Several", args: []ssml.Part{&ssml.Text{Texts: []ssml.TextPart{{Text: "olia"}}, Speed: 1.5, Voice: "as"},
			&ssml.Pause{Duration: 2 * time.Second},
			&ssml.Text{Texts: []ssml.TextPart{{Text: "olia"}}, Speed: .75, Voice: "as2"}},
			want: `<speak><voice name="as"><prosody rate="75%">olia</prosody></voice>` +
				`<break time="2000ms"/>` +
				`<voice name="as2"><prosody rate="150%">olia</prosody></voice></speak>`},
		{name: "Word", args: []ssml.Part{&ssml.Text{Texts: []ssml.TextPart{{Text: "olia", Accented: "oli{a/}"}}, Speed: 1, Voice: "as"}},
			want: `<speak><voice name="as"><prosody rate="100%"><intelektika:w acc="oli{a/}">olia</intelektika:w></prosody></voice></speak>`},
		{name: "Several words", args: []ssml.Part{&ssml.Text{Texts: []ssml.TextPart{{Text: "before"}, {Text: "olia", Accented: "oli{a/}"}, {Text: "long end"}}, Speed: 1, Voice: "as"}},
			want: `<speak><voice name="as"><prosody rate="100%">before<intelektika:w acc="oli{a/}">olia</intelektika:w>long end</prosody></voice></speak>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := saveToSSMLString(tt.args); got != tt.want {
				t.Errorf("saveToSSMLString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorker_doSSML(t *testing.T) {
	tests := []struct {
		name    string
		wChars  int
		args    string
		want    []string
		wantErr bool
	}{
		{name: "splits many words", wChars: 20, args: `<speak><intelektika:w acc="dfasdf{a/}">asdfg</intelektika:w><intelektika:w acc="dfasdf{a/}">asdfg</intelektika:w>` +
			`<intelektika:w acc="dfasdf{a/}">asdfg</intelektika:w><intelektika:w acc="dfasdf{a/}">asdfg</intelektika:w></speak>`,
			want: []string{`<speak><voice name="vd"><prosody rate="75%"><intelektika:w acc="dfasdf{a/}">asdfg</intelektika:w>` +
				`<intelektika:w acc="dfasdf{a/}">asdfg</intelektika:w></prosody></voice></speak>`,
				`<speak><voice name="vd"><prosody rate="75%"><intelektika:w acc="dfasdf{a/}">asdfg</intelektika:w>` +
					`<intelektika:w acc="dfasdf{a/}">asdfg</intelektika:w></prosody></voice></speak>`},
			wantErr: false},
		{name: "splits with words", wChars: 20, args: `<speak><intelektika:w acc="asdfasdf{a/}">asdfgasdfg</intelektika:w>0123456789 0123456789</speak>`,
			want: []string{`<speak><voice name="vd"><prosody rate="75%"><intelektika:w acc="asdfasdf{a/}">asdfgasdfg</intelektika:w>0123456789</prosody></voice></speak>`,
				`<speak><voice name="vd"><prosody rate="75%"> 0123456789</prosody></voice></speak>`},
			wantErr: false},
		{name: "splits", wChars: 20, args: "<speak>0123456789 0123456789 0123456789</speak>",
			want: []string{`<speak><voice name="vd"><prosody rate="75%">0123456789 0123456789</prosody></voice></speak>`,
				`<speak><voice name="vd"><prosody rate="75%"> 0123456789</prosody></voice></speak>`},
			wantErr: false},
		{name: "no text", wChars: 20, args: "", want: nil, wantErr: true},
		{name: "no split", wChars: 20, args: "<speak>olia</speak>",
			want:    []string{`<speak><voice name="vd"><prosody rate="75%">olia</prosody></voice></speak>`},
			wantErr: false},
		{name: "add break", wChars: 20, args: "<speak>0123456789 0123456789 0123456789<p/></speak>",
			want: []string{`<speak><voice name="vd"><prosody rate="75%">0123456789 0123456789</prosody></voice></speak>`,
				`<speak><voice name="vd"><prosody rate="75%"> 0123456789</prosody></voice><break time="1250ms"/></speak>`},
			wantErr: false},
		{name: "add several to one shot", wChars: 100, args: `<speak>0123456789 0123456789<p/> 
		<voice name="oovd"><prosody rate="x-slow">0123456789</prosody></voice></speak>`,
			want: []string{`<speak><voice name="vd"><prosody rate="75%">0123456789 0123456789</prosody></voice>` +
				`<break time="1250ms"/><voice name="oovd"><prosody rate="50%">0123456789</prosody></voice></speak>`},
			wantErr: false},
		{name: "words", wChars: 100, args: `<speak><intelektika:w acc="oli{a/}">olia</intelektika:w> 
			<voice name="oovd"><prosody rate="x-slow"><intelektika:w acc="oli{a/}a">olia2</intelektika:w></prosody></voice></speak>`,
			want: []string{`<speak><voice name="vd"><prosody rate="75%"><intelektika:w acc="oli{a/}">olia</intelektika:w></prosody></voice>` +
				`<voice name="oovd"><prosody rate="50%"><intelektika:w acc="oli{a/}a">olia2</intelektika:w></prosody></voice></speak>`},
			wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worker{wantedChars: tt.wChars}
			got, err := w.doSSML(tt.args, "vd", 1.5)
			if (err != nil) != tt.wantErr {
				t.Errorf("Worker.doSSML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Worker.doSSML() = %v, want %v", got, tt.want)
			}
		})
	}
}
