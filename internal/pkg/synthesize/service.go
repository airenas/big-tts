package synthesize

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/status"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type worker interface {
	Do(context.Context, *messages.TTSMessage) error
}

type msgSender interface {
	Send(msg amessages.Message, queue, replyQueue string) error
}

type statusSaver interface {
	Save(ID string, status, err string) error
}

// ServiceData keeps data required for service work
type ServiceData struct {
	MsgSender       msgSender
	InformMsgSender msgSender
	StatusSaver     statusSaver
	UploadCh        <-chan amqp.Delivery
	SplitCh         <-chan amqp.Delivery
	SynthesizeCh    <-chan amqp.Delivery
	JoinCh          <-chan amqp.Delivery

	Splitter    worker
	Synthesizer worker
	Joiner      worker

	StopCtx context.Context
}

//return true if it can be redelivered
type prFunc func(msg *messages.TTSMessage, data *ServiceData) (bool, error)

//StartWorkerService starts the event queue listener service to listen for events
// returns
func StartWorkerService(ctx context.Context, data *ServiceData) (<-chan struct{}, error) {
	if data.UploadCh == nil {
		return nil, errors.New("no upload channel provided")
	}
	if data.SplitCh == nil {
		return nil, errors.New("no split channel provided")
	}
	if data.SynthesizeCh == nil {
		return nil, errors.New("no synthesize channel provided")
	}
	if data.JoinCh == nil {
		return nil, errors.New("no join channel provided")
	}
	if data.MsgSender == nil {
		return nil, errors.New("no msgSender")
	}
	if data.InformMsgSender == nil {
		return nil, errors.New("no inform msgSender")
	}
	if data.StatusSaver == nil {
		return nil, errors.New("no statusSaver")
	}
	if data.Splitter == nil {
		return nil, errors.New("no splitter set")
	}
	if data.Synthesizer == nil {
		return nil, errors.New("no synthesizer set")
	}
	if data.Joiner == nil {
		return nil, errors.New("no joiner set")
	}
	goapp.Log.Infof("Starting listen for messages")

	wg := &sync.WaitGroup{}

	ctxInt, cancelF := context.WithCancel(ctx)
	cf := func() {
		cancelF()
		wg.Done()
	}

	wg.Add(4)
	go listenQueue(ctxInt, data.UploadCh, listenUpload, data, cf)
	go listenQueue(ctxInt, data.SplitCh, split, data, cf)
	go listenQueue(ctxInt, data.SynthesizeCh, synthesize, data, cf)
	go listenQueue(ctxInt, data.JoinCh, join, data, cf)

	return prepareCloseCh(wg), nil
}

func prepareCloseCh(wg *sync.WaitGroup) <-chan struct{} {
	res := make(chan struct{}, 2)
	go func() {
		wg.Wait()
		close(res)
	}()
	return res
}

func listenQueue(ctx context.Context, q <-chan amqp.Delivery, f prFunc, data *ServiceData, cancelF func()) {
	defer cancelF()
	for {
		select {
		case <-ctx.Done():
			goapp.Log.Infof("Exit queue func")
			return
		case d, ok := <-q:
			{
				if !ok {
					goapp.Log.Infof("Stopped listening queue")
					return
				}
				err := processMsg(&d, f, data)
				if err != nil {
					goapp.Log.Error(err)
				}
			}
		}
	}
}

func processMsg(d *amqp.Delivery, f prFunc, data *ServiceData) error {
	var message messages.TTSMessage
	if err := json.Unmarshal(d.Body, &message); err != nil {
		d.Nack(false, false)
		return errors.Wrap(err, "can't unmarshal message "+string(d.Body))
	}
	redeliver, err := f(&message, data)
	if err != nil {
		goapp.Log.Errorf("Can't process message %s\n%s", d.MessageId, string(d.Body))
		goapp.Log.Error(err)
		select {
		case <-data.StopCtx.Done():
			goapp.Log.Infof("Cancel msg process")
			return nil
		default:
		}
		requeue := redeliver && !d.Redelivered
		if !requeue {
			err := data.StatusSaver.Save(message.ID, "", err.Error())
			if err != nil {
				goapp.Log.Error(err)
			}
			err = data.InformMsgSender.Send(newInformMessage(&message, amessages.InformType_Failed), messages.Inform, "")
			if err != nil {
				goapp.Log.Error(err)
			}
		}
		return d.Nack(false, requeue) // redeliver for first time
	}
	return d.Ack(false)
}

//synthesize starts the synthesize process
// workflow:
// 1. set status to WORKING
// 2. send inform msg
// 3. Send split msg
func listenUpload(message *messages.TTSMessage, data *ServiceData) (bool, error) {
	goapp.Log.Infof("Got %s msg :%s", messages.Upload, message.ID)
	err := data.StatusSaver.Save(message.ID, status.Uploaded.String(), "")
	if err != nil {
		return true, err
	}
	err = data.InformMsgSender.Send(newInformMessage(message, amessages.InformType_Started), messages.Inform, "")
	if err != nil {
		return true, err
	}
	return true, data.MsgSender.Send(messages.NewMessageFrom(message), messages.Split, "")
}

func split(message *messages.TTSMessage, data *ServiceData) (bool, error) {
	goapp.Log.Infof("Got %s msg :%s", messages.Split, message.ID)
	err := data.StatusSaver.Save(message.ID, status.Split.String(), "")
	if err != nil {
		return true, err
	}
	resMsg := messages.NewMessageFrom(message)
	err = data.Splitter.Do(data.StopCtx, message)
	if err != nil {
		return true, err
	}
	return true, data.MsgSender.Send(resMsg, messages.Synthesize, "")
}

func synthesize(message *messages.TTSMessage, data *ServiceData) (bool, error) {
	goapp.Log.Infof("Got %s msg :%s", messages.Synthesize, message.ID)
	err := data.StatusSaver.Save(message.ID, status.Synthesize.String(), "")
	if err != nil {
		return true, err
	}
	resMsg := messages.NewMessageFrom(message)
	err = data.Synthesizer.Do(data.StopCtx, message)
	if err != nil {
		return true, err
	}
	return true, data.MsgSender.Send(resMsg, messages.Join, "")
}

func join(message *messages.TTSMessage, data *ServiceData) (bool, error) {
	goapp.Log.Infof("Got %s msg :%s", messages.Join, message.ID)
	err := data.StatusSaver.Save(message.ID, status.Join.String(), "")
	if err != nil {
		return true, err
	}
	err = data.Joiner.Do(data.StopCtx, message)
	if err != nil {
		return true, err
	}
	err = data.StatusSaver.Save(message.ID, status.Completed.String(), "")
	if err != nil {
		return true, err
	}
	return true, data.InformMsgSender.Send(newInformMessage(message, amessages.InformType_Finished), messages.Inform, "")
}

func newInformMessage(msg *messages.TTSMessage, it string) *amessages.InformMessage {
	return &amessages.InformMessage{QueueMessage: amessages.QueueMessage{ID: msg.ID, Tags: msg.Tags},
		Type: it, At: time.Now().UTC()}
}
