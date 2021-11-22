package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/airenas/async-api/pkg/rabbit"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/synthesize"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/labstack/gommon/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

func main() {
	goapp.StartWithDefault()

	data := &synthesize.ServiceData{}

	msgChannelProvider, err := rabbit.NewChannelProvider(goapp.Config.GetString("messageServer.url"),
		goapp.Config.GetString("messageServer.user"), goapp.Config.GetString("messageServer.pass"))
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

	if data.UploadCh, err = makeQChannel(ch, msgChannelProvider.QueueName(messages.Upload)); err != nil {
		goapp.Log.Fatal(err)
	}
	if data.SplitCh, err = makeQChannel(ch, msgChannelProvider.QueueName(messages.Split)); err != nil {
		goapp.Log.Fatal(err)
	}
	if data.SynthesizeCh, err = makeQChannel(ch, msgChannelProvider.QueueName(messages.Synthesize)); err != nil {
		goapp.Log.Fatal(err)
	}

	data.MsgSender = rabbit.NewSender(msgChannelProvider)

	printBanner()

	ctx, cancelFunc := context.WithCancel(context.Background())
	doneCh, err := synthesize.StartWorkerService(ctx, data)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't start worker service"))
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
	case <-time.After(time.Second * 10):
		logrus.Warn("Timeout gracefull shutdown")
	}
}

func initQueues(prv *rabbit.ChannelProvider) error {
	goapp.Log.Info("Initializing queues")
	for _, n := range [...]string{messages.Split, messages.Synthesize, messages.Upload, messages.Join} {
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
