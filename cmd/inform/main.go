package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	ainform "github.com/airenas/async-api/pkg/inform"
	mng "github.com/airenas/async-api/pkg/mongo"
	"github.com/airenas/async-api/pkg/rabbit"
	"github.com/airenas/big-tts/internal/pkg/inform"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/mongo"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/labstack/gommon/color"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

func main() {
	goapp.StartWithDefault()

	data := &inform.ServiceData{}
	cfg := goapp.Config

	msgChannelProvider, err := rabbit.NewChannelProvider(cfg.GetString("messageServer.url"),
		cfg.GetString("messageServer.user"), cfg.GetString("messageServer.pass"))
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init rabbitmq channel provider"))
	}
	defer msgChannelProvider.Close()
	err = initQueues(msgChannelProvider)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init queues"))
	}

	ch, err := msgChannelProvider.Channel()
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't open channel"))
	}
	if err = ch.Qos(1, 0, false); err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't set Qos"))
	}

	if data.WorkCh, err = makeQChannel(ch, msgChannelProvider.QueueName(messages.Inform)); err != nil {
		goapp.Log.Fatal(err)
	}

	mongoSessionProvider, err := mng.NewSessionProvider(cfg.GetString("mongo.url"), mongo.GetIndexes(), "tts")
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo session provider"))
	}
	defer mongoSessionProvider.Close()

	data.EmailMaker, err = ainform.NewTemplateEmailMaker(cfg)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init email maker"))
	}

	data.TaskName = cfg.GetString("worker.taskName")
	location := cfg.GetString("worker.location")
	if location != "" {
		data.Location, err = time.LoadLocation(location)
		if err != nil {
			goapp.Log.Fatal(errors.Wrap(err, "can't init location"))
		}
	}

	data.EmailSender, err = ainform.NewSimpleEmailSender(cfg)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init email sender"))
	}

	data.Locker, err = mng.NewLocker(mongoSessionProvider, mongo.EmailTable)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo locker"))
	}

	data.EmailRetriever, err = mongo.NewRequest(mongoSessionProvider)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init email retriever"))
	}

	printBanner()

	ctx, cancelFunc := context.WithCancel(context.Background())
	doneCh, err := inform.StartWorkerService(ctx, data)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't start inform worker service"))
	}
	/////////////////////// Waiting for terminate
	waitCh := make(chan os.Signal, 2)
	signal.Notify(waitCh, os.Interrupt, syscall.SIGTERM)
	select {
	case <-waitCh:
		goapp.Log.Info("Got exit signal")
	case <-doneCh:
		goapp.Log.Info("Service exit")
	}
	cancelFunc()
	select {
	case <-doneCh:
		goapp.Log.Info("All code returned. Now exit. Bye")
	case <-time.After(time.Second * 15):
		goapp.Log.Warn("Timeout gracefull shutdown")
	}
}

func initQueues(prv *rabbit.ChannelProvider) error {
	goapp.Log.Info("Initializing queues")
	for _, n := range [...]string{messages.Inform} {
		err := prv.RunOnChannelWithRetry(func(ch *amqp.Channel) error {
			_, err := rabbit.DeclareQueue(ch, prv.QueueName(n))
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func makeQChannel(ch *amqp.Channel, qname string) (<-chan amqp.Delivery, error) {
	result, err := rabbit.NewChannel(ch, qname)
	if err != nil {
		return nil, errors.Wrapf(err, "can't listen %s channel", qname)
	}
	return result, nil
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
                                   
  //|_       ____                    //|
 |/|(_)___  / __/___  _________ ___ |/||
   / / __ \/ /_/ __ \/ ___/ __ ` + "`" + `__ \    
  / / / / / __/ /_/ / /  / / / / / /    
 /_/_/ /_/_/  \____/_/  /_/ /_/ /_/  v: %s  

%s
________________________________________________________

`
	cl := color.New()
	cl.Printf(banner, cl.Red(version), cl.Green("https://github.com/airenas/big-tts"))
}
