package inform

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/test/mocks"
	"github.com/airenas/big-tts/internal/pkg/test/mocks/matchers"
	"github.com/petergtz/pegomock"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

var (
	tData    *ServiceData
	tCtx     context.Context
	tCancelF func()
	tWrkCh   chan amqp.Delivery

	tSender         *mocks.MockSender
	tEmailMaker     *mocks.MockEmailMaker
	tEmailRetriever *mocks.MockEmailRetriever
	tLocker         *mocks.MockLocker
)

func initTest(t *testing.T) {
	mocks.AttachMockToTest(t)
	tCtx, tCancelF = context.WithCancel(context.Background())
	tSender = mocks.NewMockSender()
	tEmailMaker = mocks.NewMockEmailMaker()
	tEmailRetriever = mocks.NewMockEmailRetriever()
	tLocker = mocks.NewMockLocker()

	tWrkCh = make(chan amqp.Delivery)

	tData = &ServiceData{WorkCh: tWrkCh, TaskName: "olia", Location: time.Local, EmailSender: tSender,
		EmailMaker: tEmailMaker, EmailRetriever: tEmailRetriever, Locker: tLocker}
}

func Test_Exits(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	assert.Nil(t, err)

	tCancelF()
	waitT(t, ch)
}

func waitT(t *testing.T, ch <-chan struct{}) {
	select {
	case <-ch:
	case <-time.After(time.Second * 1):
		t.Error("timeout exit")
	}
}

func Test_ExitsQueue(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	assert.Nil(t, err)
	close(tWrkCh)
	defer tCancelF()
	waitT(t, ch)
}

func Test_WorkMsg(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	assert.Nil(t, err)

	msg := amessages.InformMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, At: time.Now(), Type: amessages.InformTypeStarted}
	msgdata, _ := json.Marshal(msg)

	tWrkCh <- amqp.Delivery{Body: msgdata}
	close(tWrkCh)
	waitT(t, ch)

	tEmailRetriever.VerifyWasCalledOnce().GetEmail(pegomock.AnyString())
	tEmailMaker.VerifyWasCalledOnce().Make(matchers.AnyPtrToInformData())
	gLockID, gLockType := tLocker.VerifyWasCalledOnce().Lock(pegomock.AnyString(), pegomock.AnyString()).GetCapturedArguments()
	assert.Equal(t, "olia", gLockID)
	assert.Equal(t, amessages.InformTypeStarted, gLockType)
	gUnlockID, gUnlockType, gUnlockValue := tLocker.VerifyWasCalledOnce().UnLock(pegomock.AnyString(), pegomock.AnyString(), matchers.AnyPtrToInt()).GetCapturedArguments()
	assert.Equal(t, "olia", gUnlockID)
	assert.Equal(t, amessages.InformTypeStarted, gUnlockType)
	assert.Equal(t, 2, *gUnlockValue)
	tSender.VerifyWasCalledOnce().Send(matchers.AnyPtrToEmailEmail())
}

func Test_WorkMsg_FailRetriever(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	assert.Nil(t, err)

	msg := amessages.InformMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, At: time.Now(), Type: amessages.InformTypeStarted}
	msgdata, _ := json.Marshal(msg)

	pegomock.When(tEmailRetriever.GetEmail(pegomock.AnyString())).ThenReturn("", errors.New("err"))
	tWrkCh <- amqp.Delivery{Body: msgdata}
	close(tWrkCh)
	waitT(t, ch)

	tEmailRetriever.VerifyWasCalledOnce().GetEmail(pegomock.AnyString())
	tEmailMaker.VerifyWasCalled(pegomock.Never()).Make(matchers.AnyPtrToInformData())
	tLocker.VerifyWasCalled(pegomock.Never()).Lock(pegomock.AnyString(), pegomock.AnyString())
	tLocker.VerifyWasCalled(pegomock.Never()).UnLock(pegomock.AnyString(), pegomock.AnyString(), matchers.AnyPtrToInt())
	tSender.VerifyWasCalled(pegomock.Never()).Send(matchers.AnyPtrToEmailEmail())
}

func Test_WorkMsg_FailMaker(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	assert.Nil(t, err)

	msg := amessages.InformMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, At: time.Now(), Type: amessages.InformTypeStarted}
	msgdata, _ := json.Marshal(msg)

	pegomock.When(tEmailMaker.Make(matchers.AnyPtrToInformData())).ThenReturn(nil, errors.New("err"))
	tWrkCh <- amqp.Delivery{Body: msgdata}
	close(tWrkCh)
	waitT(t, ch)

	tEmailRetriever.VerifyWasCalledOnce().GetEmail(pegomock.AnyString())
	tEmailMaker.VerifyWasCalledOnce().Make(matchers.AnyPtrToInformData())
	tLocker.VerifyWasCalled(pegomock.Never()).Lock(pegomock.AnyString(), pegomock.AnyString())
	tLocker.VerifyWasCalled(pegomock.Never()).UnLock(pegomock.AnyString(), pegomock.AnyString(), matchers.AnyPtrToInt())
	tSender.VerifyWasCalled(pegomock.Never()).Send(matchers.AnyPtrToEmailEmail())
}

func Test_WorkMsg_FailLocker(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	assert.Nil(t, err)

	msg := amessages.InformMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, At: time.Now(), Type: amessages.InformTypeStarted}
	msgdata, _ := json.Marshal(msg)

	pegomock.When(tLocker.Lock(pegomock.AnyString(), pegomock.AnyString())).ThenReturn(errors.New("err"))
	tWrkCh <- amqp.Delivery{Body: msgdata}
	close(tWrkCh)
	waitT(t, ch)

	tEmailRetriever.VerifyWasCalledOnce().GetEmail(pegomock.AnyString())
	tEmailMaker.VerifyWasCalledOnce().Make(matchers.AnyPtrToInformData())
	tLocker.VerifyWasCalledOnce().Lock(pegomock.AnyString(), pegomock.AnyString())
	tLocker.VerifyWasCalled(pegomock.Never()).UnLock(pegomock.AnyString(), pegomock.AnyString(), matchers.AnyPtrToInt())
	tSender.VerifyWasCalled(pegomock.Never()).Send(matchers.AnyPtrToEmailEmail())
}

func Test_WorkMsg_FailSender(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	assert.Nil(t, err)

	msg := amessages.InformMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, At: time.Now(), Type: amessages.InformTypeStarted}
	msgdata, _ := json.Marshal(msg)

	pegomock.When(tSender.Send(matchers.AnyPtrToEmailEmail())).ThenReturn(errors.New("err"))
	tWrkCh <- amqp.Delivery{Body: msgdata}
	close(tWrkCh)
	waitT(t, ch)

	tEmailRetriever.VerifyWasCalledOnce().GetEmail(pegomock.AnyString())
	tEmailMaker.VerifyWasCalledOnce().Make(matchers.AnyPtrToInformData())
	tLocker.VerifyWasCalledOnce().Lock(pegomock.AnyString(), pegomock.AnyString())
	gUnlockID, gUnlockType, gUnlockValue := tLocker.VerifyWasCalledOnce().UnLock(pegomock.AnyString(), pegomock.AnyString(), matchers.AnyPtrToInt()).GetCapturedArguments()
	assert.Equal(t, "olia", gUnlockID)
	assert.Equal(t, amessages.InformTypeStarted, gUnlockType)
	assert.Equal(t, 0, *gUnlockValue)
	tSender.VerifyWasCalledOnce().Send(matchers.AnyPtrToEmailEmail())
}

func Test_validate(t *testing.T) {
	tests := []struct {
		name    string
		args    func(*ServiceData)
		wantErr bool
	}{
		{name: "OK", args: func(sd *ServiceData) {}, wantErr: false},
		{name: "OK, no Locations", args: func(sd *ServiceData) { sd.Location = nil }, wantErr: false},
		{name: "Fail", args: func(sd *ServiceData) { sd.EmailMaker = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.EmailRetriever = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.EmailSender = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.Locker = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.TaskName = "" }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.WorkCh = nil }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &ServiceData{TaskName: "olia", WorkCh: make(<-chan amqp.Delivery), EmailSender: mocks.NewMockSender(),
				EmailMaker: mocks.NewMockEmailMaker(), EmailRetriever: mocks.NewMockEmailRetriever(), Locker: mocks.NewMockLocker(),
				Location: time.Local}
			tt.args(d)
			if err := validate(d); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
