package utils

import (
	"os"

	"github.com/airenas/go-app/pkg/goapp"
)

//WriteFile write file to disk
func WriteFile(name string, data []byte) error {
	goapp.Log.Infof("Save %s", name)
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

//FileExists check if file exists
func FileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
