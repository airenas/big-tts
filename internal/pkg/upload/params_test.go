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
