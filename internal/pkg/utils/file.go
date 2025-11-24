package utils

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"
)

// WriteFile write file to disk
func WriteFile(ctx context.Context, name string, data []byte) error {
	log.Ctx(ctx).Info().Msgf("Save %s", name)
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// FileExists check if file exists
func FileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
