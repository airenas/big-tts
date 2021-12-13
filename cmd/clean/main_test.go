package main

import (
	"testing"

	amongo "github.com/airenas/async-api/pkg/mongo"
	"github.com/spf13/viper"
)

func Test_getDbCleaners(t *testing.T) {
	type args struct {
		msp *amongo.SessionProvider
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{name: "OK", args: args{msp: &amongo.SessionProvider{}}, want: 3, wantErr: false},
		{name: "Fails", args: args{msp: nil}, want: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDbCleaners(tt.args.msp)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDbCleaners() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != len(got) {
				t.Errorf("getDbCleaners() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getFileCleaners(t *testing.T) {
	type args struct {
		patterns []string
		path     string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{name: "OK", args: args{patterns: []string{"{ID}"}, path: "path"}, want: 1, wantErr: false},
		{name: "OK", args: args{patterns: []string{"{ID}", "{ID}/1"}, path: "path"}, want: 2, wantErr: false},
		{name: "OK", args: args{patterns: []string{"{ID}", "{}/1"}, path: "path"}, want: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.Set("fileStorage.patterns", tt.args.patterns)
			v.Set("fileStorage.path", tt.args.path)
			got, err := getFileCleaners(v)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFileCleaners() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("getFileCleaners() = %v, want %v", len(got), tt.want)
			}
		})
	}
}
