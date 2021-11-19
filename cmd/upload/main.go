package main

import (
	"github.com/airenas/big-tts/internal/pkg/upload"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
)

func main() {
	goapp.StartWithDefault()

	data := &upload.Data{}
	data.Port = goapp.Config.GetInt("port")

	err := upload.StartWebServer(data)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't start web server"))
	}
}
