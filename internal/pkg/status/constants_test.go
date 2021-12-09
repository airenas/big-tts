package status

import (
	"testing"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name string
		st   Status
		want string
	}{
		{st: Uploaded, want: "UPLOADED"},
		{st: Completed, want: "COMPLETED"},
		{st: Synthesize, want: "Synthesize"},
		{st: Split, want: "Split"},
		{st: Join, want: "Join"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.st.String(); got != tt.want {
				t.Errorf("Status.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFrom(t *testing.T) {
	tests := []struct {
		name string
		args string
		want Status
	}{
		{args: "COMPLETED", want: Completed},
		{args: "olia", want: 0},
		{args: "Join", want: Join},
		{args: "UPLOADED", want: Uploaded},
		{args: "Synthesize", want: Synthesize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := From(tt.args); got != tt.want {
				t.Errorf("From() = %v, want %v", got, tt.want)
			}
		})
	}
}
