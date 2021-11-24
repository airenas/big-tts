package messages

import "testing"

func TestFrom(t *testing.T) {
	type args struct {
		st string
	}
	tests := []struct {
		name string
		args args
		want TagsType
	}{
		{args: args{st: "Format"}, want: Format},
		{args: args{st: "Speed"}, want: Speed},
		{args: args{st: "Voice"}, want: Voice},
		{args: args{st: "aa"}, want: Undefined},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := From(tt.args.st); got != tt.want {
				t.Errorf("From() = %v, want %v", got, tt.want)
			}
		})
	}
}
