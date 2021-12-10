package main

import (
	"reflect"
	"testing"

	amongo "github.com/airenas/async-api/pkg/mongo"
	"github.com/airenas/big-tts/internal/pkg/clean"
	"github.com/spf13/viper"
)

func Test_getDbCleaners(t *testing.T) {
	type args struct {
		msp *amongo.SessionProvider
	}
	tests := []struct {
		name    string
		args    args
		want    []clean.Cleaner
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDbCleaners(tt.args.msp)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDbCleaners() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDbCleaners() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getFileCleaners(t *testing.T) {
	type args struct {
		cfg *viper.Viper
	}
	tests := []struct {
		name    string
		args    args
		want    []clean.Cleaner
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFileCleaners(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFileCleaners() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getFileCleaners() = %v, want %v", got, tt.want)
			}
		})
	}
}
