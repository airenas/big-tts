package synthesize

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/status"
	"github.com/airenas/big-tts/internal/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

// Worker wraps some work functionality
type Worker interface {
	Do(context.Context, *messages.TTSMessage) error
}

// MsgSender sends messages
type MsgSender interface {
	Send(msg amessages.Message, queue, replyQueue string) error
}

// StatusSaver persists data to DB
type StatusSaver interface {
	Save(ctx context.Context, ID string, status, err string) error
}

// ServiceData keeps data required for service work
type ServiceData struct {
	MsgSender       MsgSender
	InformMsgSender MsgSender
	StatusSaver     StatusSaver
	UploadCh        <-chan amqp.Delivery
	SplitCh         <-chan amqp.Delivery
	SynthesizeCh    <-chan amqp.Delivery
	JoinCh          <-chan amqp.Delivery
	RestoreUsageCh  <-chan amqp.Delivery

	Splitter      Worker
	Synthesizer   Worker
	Joiner        Worker
	UsageRestorer Worker

	// StopCtx context.Context
}

// return true if it can be redelivered
type prFunc func(ctx context.Context, msg *messages.TTSMessage, data *ServiceData) (bool, error)

// StartWorkerService starts the event queue listener service to listen for events
// returns channel for tracking if all jobs are finished
func StartWorkerService(ctx context.Context, data *ServiceData) (<-chan struct{}, error) {
	if err := validate(data); err != nil {
		return nil, err
	}
	log.Ctx(ctx).Info().Msgf("Starting listen for messages")

	wg := &sync.WaitGroup{}

	ctxInt, cancelF := context.WithCancel(ctx)
	cf := func() {
		cancelF()
		wg.Done()
	}

	wg.Add(5)
	go listenQueue(ctxInt, data.UploadCh, listenUpload, data, cf)
	go listenQueue(ctxInt, data.SplitCh, split, data, cf)
	go listenQueue(ctxInt, data.SynthesizeCh, synthesize, data, cf)
	go listenQueue(ctxInt, data.JoinCh, join, data, cf)
	go listenQueue(ctxInt, data.RestoreUsageCh, restoreUsage, data, cf)

	return prepareCloseCh(wg), nil
}

func validate(data *ServiceData) error {
	if data.UploadCh == nil {
		return errors.New("no upload channel provided")
	}
	if data.SplitCh == nil {
		return errors.New("no split channel provided")
	}
	if data.SynthesizeCh == nil {
		return errors.New("no synthesize channel provided")
	}
	if data.JoinCh == nil {
		return errors.New("no join channel provided")
	}
	if data.RestoreUsageCh == nil {
		return errors.New("no restore usage channel provided")
	}
	if data.MsgSender == nil {
		return errors.New("no msgSender")
	}
	if data.InformMsgSender == nil {
		return errors.New("no inform msgSender")
	}
	if data.StatusSaver == nil {
		return errors.New("no statusSaver")
	}
	if data.Splitter == nil {
		return errors.New("no splitter set")
	}
	if data.Synthesizer == nil {
		return errors.New("no synthesizer set")
	}
	if data.Joiner == nil {
		return errors.New("no joiner set")
	}
	if data.UsageRestorer == nil {
		return errors.New("no usage restorer set")
	}
	return nil
}

func prepareCloseCh(wg *sync.WaitGroup) <-chan struct{} {
	res := make(chan struct{})
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
			log.Ctx(ctx).Info().Msgf("Exit queue func")
			return
		case d, ok := <-q:
			{
				if !ok {
					log.Ctx(ctx).Info().Msgf("Stopped listening queue")
					return
				}
				err := processMsg(ctx, &d, f, data)
				if err != nil {
					log.Ctx(ctx).Err(err).Send()
				}
			}
		}
	}
}

func processMsg(ctx context.Context, d *amqp.Delivery, f prFunc, data *ServiceData) error {
	log.Ctx(ctx).Info().Msgf("Got msg: %s", d.RoutingKey)
	var message messages.TTSMessage
	if err := json.Unmarshal(d.Body, &message); err != nil {
		_ = d.Nack(false, false)
		return errors.Wrap(err, "can't unmarshal message "+string(d.Body))
	}
	redeliver, err := f(ctx, &message, data)
	if err != nil {
		log.Ctx(ctx).Err(err).Send()
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Info().Msgf("Cancel msg process")
			return nil
		default:
		}
		requeue := redeliver && !d.Redelivered
		if !requeue {
			errInt := data.StatusSaver.Save(ctx, message.ID, "", err.Error())
			if errInt != nil {
				log.Error().Err(errInt).Send()
			}
			errInt = data.InformMsgSender.Send(newInformMessage(&message, amessages.InformTypeFailed), messages.Inform, "")
			if errInt != nil {
				log.Error().Err(errInt).Send()
			}
			if needToRestoreUsage(err) && d.RoutingKey != messages.Fail && message.Error == "" {
				failMsg := messages.NewMessageFrom(&message)
				failMsg.Error = err.Error()
				err = data.MsgSender.Send(failMsg, messages.Fail, "")
				if err != nil {
					log.Ctx(ctx).Err(err).Send()
				}
			} else {
				log.Ctx(ctx).Info().Msg("NonRestorableError - do not send msg for restoring usage")
			}
		}
		return d.Nack(false, requeue) // redeliver for first time
	}
	return d.Ack(false)
}

func needToRestoreUsage(err error) bool {
	var errTest *utils.ErrNonRestorableUsage
	return !errors.As(err, &errTest)
}

// synthesize starts the synthesize process
// workflow:
// 1. set status to WORKING
// 2. send inform msg
// 3. Send split msg
func listenUpload(ctx context.Context, message *messages.TTSMessage, data *ServiceData) (bool, error) {
	log.Ctx(ctx).Info().Msgf("Got %s msg :%s", messages.Upload, message.ID)
	err := data.StatusSaver.Save(ctx, message.ID, status.Uploaded.String(), "")
	if err != nil {
		return true, err
	}
	err = data.InformMsgSender.Send(newInformMessage(message, amessages.InformTypeStarted), messages.Inform, "")
	if err != nil {
		return true, err
	}
	return true, data.MsgSender.Send(messages.NewMessageFrom(message), messages.Split, "")
}

func split(ctx context.Context, message *messages.TTSMessage, data *ServiceData) (bool, error) {
	log.Ctx(ctx).Info().Msgf("Got %s msg :%s", messages.Split, message.ID)
	err := data.StatusSaver.Save(ctx, message.ID, status.Split.String(), "")
	if err != nil {
		return true, err
	}
	resMsg := messages.NewMessageFrom(message)
	err = data.Splitter.Do(ctx, message)
	if err != nil {
		return true, err
	}
	return true, data.MsgSender.Send(resMsg, messages.Synthesize, "")
}

func synthesize(ctx context.Context, message *messages.TTSMessage, data *ServiceData) (bool, error) {
	log.Ctx(ctx).Info().Msgf("Got %s msg :%s", messages.Synthesize, message.ID)
	err := data.StatusSaver.Save(ctx, message.ID, status.Synthesize.String(), "")
	if err != nil {
		return true, err
	}
	resMsg := messages.NewMessageFrom(message)
	err = data.Synthesizer.Do(ctx, message)
	if err != nil {
		return true, err
	}
	return true, data.MsgSender.Send(resMsg, messages.Join, "")
}

func join(ctx context.Context, message *messages.TTSMessage, data *ServiceData) (bool, error) {
	log.Ctx(ctx).Info().Msgf("Got %s msg :%s", messages.Join, message.ID)
	err := data.StatusSaver.Save(ctx, message.ID, status.Join.String(), "")
	if err != nil {
		return true, err
	}
	err = data.Joiner.Do(ctx, message)
	if err != nil {
		return true, err
	}
	err = data.StatusSaver.Save(ctx, message.ID, status.Completed.String(), "")
	if err != nil {
		return true, err
	}
	return true, data.InformMsgSender.Send(newInformMessage(message, amessages.InformTypeFinished), messages.Inform, "")
}

func restoreUsage(ctx context.Context, message *messages.TTSMessage, data *ServiceData) (bool, error) {
	log.Ctx(ctx).Info().Msgf("Got %s msg :%s", messages.Fail, message.ID)
	return true, data.UsageRestorer.Do(ctx, message)
}

func newInformMessage(msg *messages.TTSMessage, it string) *amessages.InformMessage {
	return &amessages.InformMessage{QueueMessage: amessages.QueueMessage{ID: msg.ID, Tags: msg.Tags},
		Type: it, At: time.Now().UTC()}
}
