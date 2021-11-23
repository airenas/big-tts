package utils

import "os"

func WriteFile(name string, data []byte) error {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func FileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
