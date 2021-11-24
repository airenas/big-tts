package splitter

import (
	"testing"
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
