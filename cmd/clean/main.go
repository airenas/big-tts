package main

import (
	aclean "github.com/airenas/async-api/pkg/clean"
	amongo "github.com/airenas/async-api/pkg/mongo"
	"github.com/spf13/viper"

	"github.com/airenas/big-tts/internal/pkg/clean"
	"github.com/airenas/big-tts/internal/pkg/mongo"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/labstack/gommon/color"
	"github.com/pkg/errors"
)

func main() {
	goapp.StartWithDefault()
	cfg := goapp.Config
	data := &clean.Data{}
	data.Port = cfg.GetInt("port")
	var err error

	cleaner := &aclean.CleanerGroup{}

	mongoSessionProvider, err := amongo.NewSessionProvider(cfg.GetString("mongo.url"), mongo.GetIndexes(), "tts")
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo session provider"))
	}
	defer mongoSessionProvider.Close()

	cls, err := getDbCleaners(mongoSessionProvider)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo table cleaners"))
	}
	for _, cl := range cls {
		cleaner.Jobs = append(cleaner.Jobs, cl)
	}

	cls, err = getFileCleaners(cfg)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init file cleaners"))
	}
	for _, cl := range cls {
		cleaner.Jobs = append(cleaner.Jobs, cl)
	}

	data.Cleaner = cleaner

	printBanner()

	err = clean.StartWebServer(data)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't start web server"))
	}
}

func getDbCleaners(msp *amongo.SessionProvider) ([]clean.Cleaner, error) {
	res := make([]clean.Cleaner, 0)
	for _, t := range mongo.Tables() {
		cl, err := amongo.NewCleanRecord(msp, t)
		if err != nil {
			return nil, errors.Wrapf(err, "can't init cleaner for table %s", t)
		}
		res = append(res, cl)
	}
	return res, nil
}

func getFileCleaners(cfg *viper.Viper) ([]clean.Cleaner, error) {
	patterns := cfg.GetStringSlice("fileStorage.patterns")
	path := cfg.GetString("fileStorage.path")
	res := make([]clean.Cleaner, 0)
	for _, p := range patterns {
		cl, err := aclean.NewLocalFile(path, p)
		if err != nil {
			return nil, errors.Wrapf(err, "can't init cleaner for path %s, %s", path, p)
		}
		res = append(res, cl)
	}
	return res, nil
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
                                   
        __                                        
  _____/ /__  ____ _____           _  __          
 / ___/ / _ \/ __ ` + "`" + `/ __ \   ______| |/_/_____     
/ /__/ /  __/ /_/ / / / /  /_____/>  </_____/     
\___/_/\___/\__,_/_/ /_/        /_/|_| v: %s    

%s
________________________________________________________                                                 

`
	cl := color.New()
	cl.Printf(banner, cl.Red(version), cl.Green("https://github.com/airenas/big-tts"))
}
