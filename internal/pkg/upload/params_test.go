package upload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getSpeed(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    float64
		wantErr bool
	}{
		{args: "0", want: 0, wantErr: false},
		{args: "-20", want: 0, wantErr: true},
		{args: "0.4999", want: 0, wantErr: true},
		{args: "0.5", want: 0.5, wantErr: false},
		{args: "2", want: 2, wantErr: false},
		{args: "1", want: 1, wantErr: false},
		{args: "", want: 1, wantErr: false},
		{args: "aaa", want: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := getSpeed(tt.args)
			assert.InDelta(t, tt.want, v, 0.0001, "fail")
			assert.Equal(t, tt.wantErr, err != nil, "fail - %v", err)
		})
	}
}

func Test_getAllowCollect(t *testing.T) {
	type args struct {
		v *bool
		s string
	}
	trueV := true
	falseV := false
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{name: "Request", args: args{v: &trueV, s: ""}, want: true, wantErr: false},
		{name: "Request", args: args{v: &trueV, s: "request"}, want: true, wantErr: false},
		{name: "Request", args: args{v: &falseV, s: ""}, want: false, wantErr: false},
		{name: "Request", args: args{v: &falseV, s: "request"}, want: false, wantErr: false},
		{name: "Always", args: args{v: &falseV, s: "always"}, want: false, wantErr: true},
		{name: "Never", args: args{v: &trueV, s: "never"}, want: false, wantErr: true},
		{name: "Always", args: args{v: nil, s: "always"}, want: true, wantErr: false},
		{name: "Never", args: args{v: nil, s: "never"}, want: false, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAllowCollect(tt.args.v, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAllowCollect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getAllowCollect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getOutputAudioFormat(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    string
		wantErr bool
	}{
		{name: "mp3", args: "mp3", want: "mp3", wantErr: false},
		{name: "m4a", args: "m4a", want: "m4a", wantErr: false},
		{name: "Empty", args: "", want: "", wantErr: false},
		{name: "Err", args: "wav", want: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getOutputAudioFormat(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("getOutputAudioFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getOutputAudioFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}
