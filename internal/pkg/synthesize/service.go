package synthesize

import (
	"context"
	"encoding/json"
	"sync"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type msgSender interface {
	Send(msg amessages.Message, queue, replyQueue string) error
}

// ServiceData keeps data required for service work
type ServiceData struct {
	MsgSender    msgSender
	UploadCh     <-chan amqp.Delivery
	SplitCh      <-chan amqp.Delivery
	SynthesizeCh <-chan amqp.Delivery
}

//return true if it can be redelivered
type prFunc func(d *amqp.Delivery, data *ServiceData) (bool, error)

//StartWorkerService starts the event queue listener service to listen for events
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

	goapp.Log.Infof("Starting listen for messages")

	wg := &sync.WaitGroup{}

	go listenQueue(ctx, data.UploadCh, listenUpload, data, wg)
	go listenQueue(ctx, data.SplitCh, split, data, wg)
	go listenQueue(ctx, data.SynthesizeCh, synthesize, data, wg)

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

func listenQueue(ctx context.Context, q <-chan amqp.Delivery, f prFunc, data *ServiceData, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-q:
			{
				if !ok {
					goapp.Log.Infof("Stopped listening queue")
					return
				}
				redeliver, err := f(&d, data)
				if err != nil {
					goapp.Log.Errorf("Can't process message %s\n%s", d.MessageId, string(d.Body))
					goapp.Log.Error(err)
					d.Nack(false, redeliver && !d.Redelivered) // redeliver for first time
				} else {
					d.Ack(false)
				}
			}
		}
	}
}

//synthesize starts the synthesize process
// workflow:
// 1. set status to WORKING
// 2. send inform msg
// 3. Send split msg
func listenUpload(d *amqp.Delivery, data *ServiceData) (bool, error) {
	var message amessages.QueueMessage
	if err := json.Unmarshal(d.Body, &message); err != nil {
		return false, errors.Wrap(err, "can't unmarshal message "+string(d.Body))
	}

	goapp.Log.Infof("Got %s msg :%s", messages.Upload, message.ID)
	// err := data.StatusSaver.Save(message.ID, status.AudioConvert)
	// if err != nil {
	// 	cmdapp.Log.Error(err)
	// 	return true, err
	// }
	// err = data.InformMessageSender.Send(newInformMessage(&message, messages.InformType_Started), messages.Inform, "")
	// if err != nil {
	// 	return true, err
	// }
	return true, data.MsgSender.Send(amessages.NewQueueMessageFromM(&message), messages.Split, "")
}

func split(d *amqp.Delivery, data *ServiceData) (bool, error) {
	var message amessages.QueueMessage
	if err := json.Unmarshal(d.Body, &message); err != nil {
		return false, errors.Wrap(err, "can't unmarshal message "+string(d.Body))
	}

	goapp.Log.Infof("Got %s msg :%s", messages.Split, message.ID)
	// err := data.StatusSaver.Save(message.ID, status.AudioConvert)
	// if err != nil {
	// 	cmdapp.Log.Error(err)
	// 	return true, err
	// }
	// err = data.InformMessageSender.Send(newInformMessage(&message, messages.InformType_Started), messages.Inform, "")
	// if err != nil {
	// 	return true, err
	// }
	return true, data.MsgSender.Send(amessages.NewQueueMessageFromM(&message), messages.Synthesize, "")
}

func synthesize(d *amqp.Delivery, data *ServiceData) (bool, error) {
	var message amessages.QueueMessage
	if err := json.Unmarshal(d.Body, &message); err != nil {
		return false, errors.Wrap(err, "can't unmarshal message "+string(d.Body))
	}

	goapp.Log.Infof("Got %s msg :%s", messages.Synthesize, message.ID)
	// err := data.StatusSaver.Save(message.ID, status.AudioConvert)
	// if err != nil {
	// 	cmdapp.Log.Error(err)
	// 	return true, err
	// }
	// err = data.InformMessageSender.Send(newInformMessage(&message, messages.InformType_Started), messages.Inform, "")
	// if err != nil {
	// 	return true, err
	// }
	return true, data.MsgSender.Send(amessages.NewQueueMessageFromM(&message), messages.Join, "")
}

// func sendInformFailure(message *messages.QueueMessage, data *ServiceData) {
// 	cmdapp.Log.Infof("Trying send inform msg about failure %s", message.ID)
// 	err := data.InformMessageSender.Send(newInformMessage(message, messages.InformType_Failed), messages.Inform, "")
// 	cmdapp.LogIf(err)
// 	if tq, ok := messages.GetTag(message.Tags, messages.TagResultQueue); ok {
// 		msg := messages.NewQueueMessageFromM(message)
// 		err := data.MessageSender.Send(msg, tq, "")
// 		cmdapp.LogIf(err)
// 	}
// }

// 4. Synthesize all batches
// 5. Join audio
// 6. set status to COMPLETED
// 7. send inform msg
