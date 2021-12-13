package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	mng "github.com/airenas/async-api/pkg/mongo"
	"github.com/airenas/async-api/pkg/rabbit"
	"github.com/airenas/big-tts/internal/pkg/joiner"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/mongo"
	"github.com/airenas/big-tts/internal/pkg/splitter"
	"github.com/airenas/big-tts/internal/pkg/synthesize"
	"github.com/airenas/big-tts/internal/pkg/synthesizer"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/labstack/gommon/color"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

func main() {
	goapp.StartWithDefault()

	data := &synthesize.ServiceData{}
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

	if data.UploadCh, err = makeQChannel(ch, msgChannelProvider.QueueName(messages.Upload)); err != nil {
		goapp.Log.Fatal(err)
	}
	if data.SplitCh, err = makeQChannel(ch, msgChannelProvider.QueueName(messages.Split)); err != nil {
		goapp.Log.Fatal(err)
	}
	if data.SynthesizeCh, err = makeQChannel(ch, msgChannelProvider.QueueName(messages.Synthesize)); err != nil {
		goapp.Log.Fatal(err)
	}
	if data.JoinCh, err = makeQChannel(ch, msgChannelProvider.QueueName(messages.Join)); err != nil {
		goapp.Log.Fatal(err)
	}

	data.MsgSender = rabbit.NewSender(msgChannelProvider)
	data.InformMsgSender = data.MsgSender

	mongoSessionProvider, err := mng.NewSessionProvider(cfg.GetString("mongo.url"), mongo.GetIndexes(), "tts")
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo session provider"))
	}
	defer mongoSessionProvider.Close()

	data.StatusSaver, err = mongo.NewStatus(mongoSessionProvider)
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init mongo status saver"))
	}
	data.Splitter, err = splitter.NewWorker(cfg.GetString("splitter.inTemplate"),
		cfg.GetString("splitter.outTemplate"))
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init splitter"))
	}
	data.Synthesizer, err = synthesizer.NewWorker(cfg.GetString("splitter.outTemplate"),
		cfg.GetString("synthesizer.outTemplate"),
		cfg.GetString("synthesizer.URL"),
		cfg.GetInt("synthesizer.workers"))
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init synthesizer"))
	}
	data.Joiner, err = joiner.NewWorker(cfg.GetString("synthesizer.outTemplate"),
		cfg.GetString("joiner.outTemplate"),
		cfg.GetString("joiner.workTemplate"),
		cfg.GetStringSlice("joiner.metadata"))
	if err != nil {
		goapp.Log.Fatal(errors.Wrap(err, "can't init joiner"))
	}

	printBanner()

	ctx, cancelFunc := context.WithCancel(context.Background())
	data.StopCtx = ctx
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
		goapp.Log.Info("All code returned. Now exit. Bye")
	case <-time.After(time.Second * 15):
		goapp.Log.Warn("Timeout gracefull shutdown")
	}
}

func initQueues(prv *rabbit.ChannelProvider) error {
	goapp.Log.Info("Initializing queues")
	for _, n := range [...]string{messages.Split, messages.Synthesize, messages.Upload, messages.Join, messages.Inform} {
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
/_____/___/\____/   \__/\__/____/  v: %s  
                                   
                        __  __              _          
      _______  ______  / /_/ /_  ___  _____(_)___  ___ 
     / ___/ / / / __ \/ __/ __ \/ _ \/ ___/ /_  / / _ \
    (__  ) /_/ / / / / /_/ / / /  __(__  ) / / /_/  __/
   /____/\__, /_/ /_/\__/_/ /_/\___/____/_/ /___/\___/ 
        /____/                                         

%s
________________________________________________________

`
	cl := color.New()
	cl.Printf(banner, cl.Red(version), cl.Green("https://github.com/airenas/big-tts"))
}
