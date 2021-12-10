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
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

var (
	tData     *ServiceData
	tCtx      context.Context
	tCancelF  func()
	tUploadCh chan amqp.Delivery

	tStatusMock *mocks.MockStatusSaver
	tMsgSender  *mocks.MockMsgSender
	tInfSender  *mocks.MockMsgSender
)

func initTest(t *testing.T) {
	mocks.AttachMockToTest(t)
	tCtx, tCancelF = context.WithCancel(context.Background())
	tStatusMock = mocks.NewMockStatusSaver()
	tMsgSender = mocks.NewMockMsgSender()
	tInfSender = mocks.NewMockMsgSender()

	tUploadCh = make(chan amqp.Delivery)
	tData = &ServiceData{UploadCh: tUploadCh, SplitCh: make(<-chan amqp.Delivery),
		SynthesizeCh: make(<-chan amqp.Delivery), JoinCh: make(<-chan amqp.Delivery), MsgSender: tMsgSender,
		InformMsgSender: tInfSender, StatusSaver: tStatusMock, Splitter: mocks.NewMockWorker(),
		Synthesizer: mocks.NewMockWorker(), Joiner: mocks.NewMockWorker()}

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
	case <-time.After(time.Second):
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
	assert.Nil(t, err)

	msg := messages.TTSMessage{QueueMessage: amessages.QueueMessage{ID: "olia"}, Voice: "aa"}
	msgdata, _ := json.Marshal(msg)

	tUploadCh <- amqp.Delivery{Body: msgdata}

	tStatusMock.VerifyWasCalledOnce().Save(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString())
	tMsgSender.VerifyWasCalledOnce().Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())
	tInfSender.VerifyWasCalledOnce().Send(matchers.AnyMessagesMessage(), pegomock.AnyString(), pegomock.AnyString())

	tCancelF()
	waitT(t, ch)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &ServiceData{UploadCh: make(<-chan amqp.Delivery), SplitCh: make(<-chan amqp.Delivery),
				SynthesizeCh: make(<-chan amqp.Delivery), JoinCh: make(<-chan amqp.Delivery), MsgSender: mocks.NewMockMsgSender(),
				InformMsgSender: mocks.NewMockMsgSender(), StatusSaver: mocks.NewMockStatusSaver(), Splitter: mocks.NewMockWorker(),
				Synthesizer: mocks.NewMockWorker(), Joiner: mocks.NewMockWorker()}
			tt.args(d)
			if err := validate(d); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
