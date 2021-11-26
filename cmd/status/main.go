package main

import (
	mng "github.com/airenas/async-api/pkg/mongo"

	"github.com/airenas/big-tts/internal/pkg/mongo"
	"github.com/airenas/big-tts/internal/pkg/statusservice"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/labstack/gommon/color"
	"github.com/pkg/errors"
)

func main() {
	goapp.StartWithDefault()
	cfg := goapp.Config
	data := &statusservice.Data{}
	data.Port = cfg.GetInt("port")
	var err error

	mongoSessionProvider, err := mng.NewSessionProvider(cfg.GetString("mongo.url"), mongo.GetIndexes(), "tts")
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo session provider"))
	}
	defer mongoSessionProvider.Close()

	data.StatusProvider, err = mongo.NewStatus(mongoSessionProvider)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo request saver"))
	}

	printBanner()

	err = statusservice.StartWebServer(data)
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
                                   
         __        __            
   _____/ /_____ _/ /___  _______
  / ___/ __/ __ ` + "`" + `/ __/ / / / ___/
 (__  ) /_/ /_/ / /_/ /_/ (__  ) 
/____/\__/\__,_/\__/\__,_/____/   v: %s    

%s
________________________________________________________                                                 

`
	cl := color.New()
	cl.Printf(banner, cl.Red(version), cl.Green("https://github.com/airenas/big-tts"))
}
