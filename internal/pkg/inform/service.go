package inform

import (
	"context"
	"encoding/json"
	"time"

	"github.com/airenas/async-api/pkg/inform"
	"github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/jordan-wright/email"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

//Sender send emails
type Sender interface {
	Send(email *email.Email) error
}

//EmailMaker prepares the email
type EmailMaker interface {
	Make(data *inform.Data) (*email.Email, error)
}

//EmailRetriever return the email by ID
type EmailRetriever interface {
	GetEmail(ID string) (string, error)
}

//Locker tracks email sending process
//It is used to quarantee not to send the emails twice
type Locker interface {
	Lock(id string, lockKey string) error
	UnLock(id string, lockKey string, value *int) error
}

// ServiceData keeps data required for service work
type ServiceData struct {
	TaskName       string
	WorkCh         <-chan amqp.Delivery
	EmailSender    Sender
	EmailMaker     EmailMaker
	EmailRetriever EmailRetriever
	Locker         Locker
	Location       *time.Location
}

//StartWorkerService starts the event queue listener service to listen for configured events
// return channel to track the finish event
//
// to wait sync for the service to finish:
// fc, err := StartWorkerService(data)
// handle err
// <-fc // waits for finish
func StartWorkerService(ctx context.Context, data *ServiceData) (<-chan struct{}, error) {
	goapp.Log.Infof("Starting listen for messages")
	if err := validate(data); err != nil {
		return nil, err
	}

	ctxInt, cancelF := context.WithCancel(context.Background())
	go listenQueue(ctx, data.WorkCh, data, cancelF)
	return ctxInt.Done(), nil
}

func validate(data *ServiceData) error {
	if data.TaskName == "" {
		return errors.New("no Task Name")
	}
	if data.EmailMaker == nil {
		return errors.New("no email maker")
	}
	if data.EmailRetriever == nil {
		return errors.New("no email retriever")
	}
	if data.EmailSender == nil {
		return errors.New("no sender")
	}
	if data.Locker == nil {
		return errors.New("no locker")
	}
	if data.WorkCh == nil {
		return errors.New("no work channel")
	}
	return nil
}

//work is main method to send the message
func work(data *ServiceData, message *messages.InformMessage) error {
	goapp.Log.Infof("Got task %s for ID: %s", data.TaskName, message.ID)

	mailData := inform.Data{}
	mailData.ID = message.ID
	mailData.MsgTime = toLocalTime(data, message.At)
	mailData.MsgType = message.Type

	var err error
	mailData.Email, err = data.EmailRetriever.GetEmail(message.ID)
	if err != nil {
		goapp.Log.Error(err)
		return errors.Wrap(err, "can't retrieve email")
	}

	email, err := data.EmailMaker.Make(&mailData)
	if err != nil {
		goapp.Log.Error(err)
		return errors.Wrap(err, "can't prepare email")
	}

	err = data.Locker.Lock(mailData.ID, mailData.MsgType)
	if err != nil {
		goapp.Log.Error(err)
		return errors.Wrap(err, "can't lock mail table")
	}
	var unlockValue = 0
	defer data.Locker.UnLock(mailData.ID, mailData.MsgType, &unlockValue)

	err = data.EmailSender.Send(email)
	if err != nil {
		goapp.Log.Error(err)
		return errors.Wrap(err, "can't send email")
	}
	unlockValue = 2
	return nil
}

func listenQueue(ctx context.Context, q <-chan amqp.Delivery, data *ServiceData, cancelF func()) {
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
				err := processMsg(ctx, &d, data)
				if err != nil {
					goapp.Log.Error(err)
				}
			}
		}
	}
}

func processMsg(ctx context.Context, d *amqp.Delivery, data *ServiceData) error {
	var message messages.InformMessage
	if err := json.Unmarshal(d.Body, &message); err != nil {
		d.Nack(false, false)
		return errors.Wrap(err, "can't unmarshal message "+string(d.Body))
	}
	err := work(data, &message)
	if err != nil {
		goapp.Log.Errorf("can't process message %s\n%s", d.MessageId, string(d.Body))
		goapp.Log.Error(err)
		select {
		case <-ctx.Done():
			goapp.Log.Infof("Cancel msg process")
			return nil
		default:
		}
		return d.Nack(false, !d.Redelivered) // redeliver for first time
	}
	return d.Ack(false)
}

func toLocalTime(data *ServiceData, t time.Time) time.Time {
	if data.Location != nil {
		return t.In(data.Location)
	}
	return t
}
