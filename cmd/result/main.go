package main

import (
	mng "github.com/airenas/async-api/pkg/mongo"

	"github.com/airenas/async-api/pkg/file"
	"github.com/airenas/big-tts/internal/pkg/result"
	"github.com/airenas/big-tts/internal/pkg/mongo"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/labstack/gommon/color"
	"github.com/pkg/errors"
)

func main() {
	goapp.StartWithDefault()
	cfg := goapp.Config
	data := &result.Data{}
	data.Port = cfg.GetInt("port")
	var err error

	data.Reader, err = file.NewLocalLoader(cfg.GetString("fileStorage.path"))
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init file storage reader"))
	}

	mongoSessionProvider, err := mng.NewSessionProvider(cfg.GetString("mongo.url"), mongo.GetIndexes(), "tts")
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo session provider"))
	}
	defer mongoSessionProvider.Close()

	data.NameProvider, err = mongo.NewRequest(mongoSessionProvider)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo request saver"))
	}

	printBanner()

	err = result.StartWebServer(data)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't start web server"))
	}
}

var (
	version = "DEV"
)

func printBanner() {
	banner := `
    ____  __________   __  __      
   / __ )/  _/ ____/  / /_/ /______
  / __  |/ // / __   / __/ __/ ___/
 / /_/ // // /_/ /  / /_/ /_(__  ) 
/_____/___/\____/   \__/\__/____/  
                                   
                         ____             __       
   ________  _______  __/ / /_            \ \      
  / ___/ _ \/ ___/ / / / / __/  ___________\ \     
 / /  /  __(__  ) /_/ / / /_   /_____/_____/ /     
/_/   \___/____/\__,_/_/\__/              /_/ v: %s    

%s
________________________________________________________                                                 

`
	cl := color.New()
	cl.Printf(banner, cl.Red(version), cl.Green("https://github.com/airenas/big-tts"))
}
