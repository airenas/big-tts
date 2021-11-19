package main

import (
	"github.com/airenas/async-api/pkg/file"
	mng "github.com/airenas/async-api/pkg/mongo"
	"github.com/airenas/big-tts/internal/pkg/mongo"
	"github.com/airenas/big-tts/internal/pkg/upload"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/labstack/gommon/color"
	"github.com/pkg/errors"
)

func main() {
	goapp.StartWithDefault()

	data := &upload.Data{}
	data.Port = goapp.Config.GetInt("port")
	var err error
	data.Saver, err = file.NewLocalSaver(goapp.Config.GetString("fileStorage.path"))
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init file saver"))
	}

	mongoSessionProvider, err := mng.NewSessionProvider(goapp.Config.GetString("mongo.url"), nil, "tts")
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo session provider"))
	}
	defer mongoSessionProvider.Close()

	data.ReqSaver, err = mongo.NewRequestSaver(mongoSessionProvider)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo request saver"))
	}

	printBanner()

	err = upload.StartWebServer(data)
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
                                   
                   __                __   __           
      __  ______  / /___  ____ _____/ /  / /           
     / / / / __ \/ / __ \/ __ ` + "`" + `/ __  /  / / ______     
    / /_/ / /_/ / / /_/ / /_/ / /_/ /   \ \/_____/     
    \__,_/ .___/_/\____/\__,_/\__,_/     \_\        v: %s    
        /_/                                    

%s
________________________________________________________                                                 

`
	cl := color.New()
	cl.Printf(banner, cl.Red(version), cl.Green("https://github.com/airenas/big-tts"))
}
