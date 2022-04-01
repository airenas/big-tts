package synthesize

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/test/mocks"
	"github.com/airenas/big-tts/internal/pkg/test/mocks/matchers"
	"github.com/petergtz/pegomock"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	tData         *ServiceData
	tCtx          context.Context
	tCancelF      func()
	tUploadCh     chan amqp.Delivery
	tSplitCh      chan amqp.Delivery
	tSynthesizeCh chan amqp.Delivery
	tJoinCh       chan amqp.Delivery
	tRestoreCh    chan amqp.Delivery

	tStatusMock    *mocks.MockStatusSaver
	tMsgSender     *mocks.MockMsgSender
	tInfSender     *mocks.MockMsgSender
	tSplitWrk      *mocks.MockWorker
	tSynthesizeWrk *mocks.MockWorker
	tJoinWrk       *mocks.MockWorker
	tRestoreWrk    *mocks.MockWorker
)

func initTest(t *testing.T) {
	t.Helper()
	mocks.AttachMockToTest(t)
	tCtx, tCancelF = context.WithCancel(context.Background())
	tStatusMock = mocks.NewMockStatusSaver()
	tMsgSender = mocks.NewMockMsgSender()
	tInfSender = mocks.NewMockMsgSender()
	tSplitWrk = mocks.NewMockWorker()
	tSynthesizeWrk = mocks.NewMockWorker()
	tJoinWrk = mocks.NewMockWorker()
	tRestoreWrk = mocks.NewMockWorker()

	tUploadCh = make(chan amqp.Delivery)
	tSplitCh = make(chan amqp.Delivery)
	tSynthesizeCh = make(chan amqp.Delivery)
	tJoinCh = make(chan amqp.Delivery)
	tRestoreCh = make(chan amqp.Delivery)

	tData = &ServiceData{UploadCh: tUploadCh, SplitCh: tSplitCh,
		SynthesizeCh: tSynthesizeCh, JoinCh: tJoinCh, MsgSender: tMsgSender,
		InformMsgSender: tInfSender, StatusSaver: tStatusMock, Splitter: tSplitWrk,
		Synthesizer: tSynthesizeWrk, Joiner: tJoinWrk,
		RestoreUsageCh: tRestoreCh, UsageRestorer: tRestoreWrk}
	tData.StopCtx = tCtx
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
	close(tUploadCh)
	defer tCancelF()
	waitT(t, ch)
}

func Test_UploadMsg(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	require.Nil(t, err)

	msg := messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, Voice: "aa"}
	msgdata, _ := json.Marshal(msg)

	tUploadCh <- amqp.Delivery{Body: msgdata}
	close(tUploadCh)
	waitT(t, ch)

	tStatusMock.VerifyWasCalledOnce().Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())
	tMsgSender.VerifyWasCalledOnce().Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())
	tInfSender.VerifyWasCalledOnce().Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())
}

func Test_UploadMsg_FailSave(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	require.Nil(t, err)
	pegomock.When(tStatusMock.Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())).ThenReturn(errors.New("err"))

	msg := messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, Voice: "aa"}
	msgdata, _ := json.Marshal(msg)
	tUploadCh <- amqp.Delivery{Body: msgdata}
	close(tUploadCh)
	waitT(t, ch)

	tStatusMock.VerifyWasCalledOnce().Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())
	tMsgSender.VerifyWasCalled(pegomock.Never()).Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())
	tInfSender.VerifyWasCalled(pegomock.Never()).Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())
}

func Test_UploadMsg_FailSave_Redelivered(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	require.Nil(t, err)
	pegomock.When(tStatusMock.Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())).ThenReturn(errors.New("err"))

	msg := messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, Voice: "aa", RequestID: "rID"}
	msgdata, _ := json.Marshal(msg)
	tUploadCh <- amqp.Delivery{Body: msgdata, Redelivered: true}
	close(tUploadCh)
	waitT(t, ch)

	tStatusMock.VerifyWasCalled(pegomock.Twice()).Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())
	fMsg, fQueue, _ := tMsgSender.VerifyWasCalled(pegomock.Once()).Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString()).
		GetCapturedArguments()
	assert.Equal(t, messages.Fail, fQueue)
	assert.Equal(t, "err", fMsg.(*messages.TTSMessage).Error)
	assert.Equal(t, "rID", fMsg.(*messages.TTSMessage).RequestID)
	_, eQueue, _ := tInfSender.VerifyWasCalled(pegomock.Once()).Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString()).
		GetCapturedArguments()
	assert.Equal(t, messages.Inform, eQueue)
}

func Test_SplitMsg(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	require.Nil(t, err)

	msg := messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, Voice: "aa"}
	msgdata, _ := json.Marshal(msg)

	tSplitCh <- amqp.Delivery{Body: msgdata}
	close(tSplitCh)
	waitT(t, ch)

	tStatusMock.VerifyWasCalledOnce().Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())
	tMsgSender.VerifyWasCalledOnce().Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())
	tSplitWrk.VerifyWasCalledOnce().Do(matchers.AnyContextContext(), matchers.AnyPtrToMessagesTTSMessage())
}

func Test_SplitMsg_Fail(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	require.Nil(t, err)

	pegomock.When(tSplitWrk.Do(matchers.AnyContextContext(), matchers.AnyPtrToMessagesTTSMessage())).ThenReturn(errors.New("err"))

	msg := messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, Voice: "aa"}
	msgdata, _ := json.Marshal(msg)
	tSplitCh <- amqp.Delivery{Body: msgdata}
	close(tSplitCh)
	waitT(t, ch)

	tStatusMock.VerifyWasCalledOnce().Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())
	tMsgSender.VerifyWasCalled(pegomock.Never()).Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())
}

func Test_SynthesizeMsg(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	require.Nil(t, err)

	msg := messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, Voice: "aa"}
	msgdata, _ := json.Marshal(msg)

	tSynthesizeCh <- amqp.Delivery{Body: msgdata}
	close(tSynthesizeCh)
	waitT(t, ch)

	tStatusMock.VerifyWasCalledOnce().Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())
	tMsgSender.VerifyWasCalledOnce().Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())
	tSynthesizeWrk.VerifyWasCalledOnce().Do(matchers.AnyContextContext(), matchers.AnyPtrToMessagesTTSMessage())
}

func Test_JoinMsg(t *testing.T) {
	initTest(t)
	ch, err := StartWorkerService(tCtx, tData)
	require.Nil(t, err)

	msg := messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, Voice: "aa"}
	msgdata, _ := json.Marshal(msg)

	tJoinCh <- amqp.Delivery{Body: msgdata}
	close(tJoinCh)
	waitT(t, ch)

	tStatusMock.VerifyWasCalled(pegomock.Twice()).Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())
	tInfSender.VerifyWasCalledOnce().Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())
	tJoinWrk.VerifyWasCalledOnce().Do(matchers.AnyContextContext(), matchers.AnyPtrToMessagesTTSMessage())
}

func Test_validate(t *testing.T) {
	tests := []struct {
		name    string
		args    func(*ServiceData)
		wantErr bool
	}{
		{name: "OK", args: func(sd *ServiceData) {}, wantErr: false},
		{name: "Fail", args: func(sd *ServiceData) { sd.UploadCh = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.SplitCh = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.SynthesizeCh = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.JoinCh = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.MsgSender = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.InformMsgSender = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.StatusSaver = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.Splitter = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.Synthesizer = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.Joiner = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.RestoreUsageCh = nil }, wantErr: true},
		{name: "Fail", args: func(sd *ServiceData) { sd.UsageRestorer = nil }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &ServiceData{UploadCh: make(<-chan amqp.Delivery), SplitCh: make(<-chan amqp.Delivery),
				SynthesizeCh: make(<-chan amqp.Delivery), JoinCh: make(<-chan amqp.Delivery),
				RestoreUsageCh: make(<-chan amqp.Delivery), UsageRestorer: mocks.NewMockWorker(),
				MsgSender:       mocks.NewMockMsgSender(),
				InformMsgSender: mocks.NewMockMsgSender(), StatusSaver: mocks.NewMockStatusSaver(), Splitter: mocks.NewMockWorker(),
				Synthesizer: mocks.NewMockWorker(), Joiner: mocks.NewMockWorker()}
			tt.args(d)
			if err := validate(d); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
