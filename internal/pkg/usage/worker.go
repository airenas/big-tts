package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
)

//Worker implements usage restore functionality
type Worker struct {
	serviceURL string
	httpClient http.Client
}

//NewWorker creates new synthesize worker
func NewWorker(url string) (*Worker, error) {
	if url == "" {
		return nil, errors.Errorf("no service URL")
	}
	res := &Worker{serviceURL: url}
	res.httpClient = http.Client{Transport: &http.Transport{
		MaxIdleConns:        5,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
		MaxConnsPerHost:     10,
	}}

	goapp.Log.Infof("Doorman-admin URL: %s", res.serviceURL)
	return res, nil
}

//Do tries to restore usage
func (w *Worker) Do(ctx context.Context, msg *messages.TTSMessage) error {
	goapp.Log.Infof("Doing usage restoratioon for %s. requestID: %s", msg.ID, msg.RequestID)
	if msg.RequestID == "" {
		goapp.Log.Warn("no requestID")
		return nil
	}
	service, rID, err := parse(msg.RequestID)
	if err != nil {
		return errors.Wrapf(err, "wrong requestID format '%s'", msg.RequestID)
	}
	return w.invoke(service, rID, msg.Error)
}

func parse(s string) (string, string, error) {
	strs := strings.SplitN(s, ":", 2)
	if len(strs) != 2 || strs[0] == "" || strs[1] == "" {
		return "", "", errors.New("wrong format, expected 'srv:manual:requestID'")
	}
	return strs[0], strs[1], nil
}

type request struct {
	Error string `json:"error,omitempty"`
}

func (w *Worker) invoke(service, requestID, errorMsg string) error {
	inp := request{Error: errorMsg}
	b, err := json.Marshal(inp)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/%s/restore/%s", w.serviceURL, service, requestID), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	ctx, cancelF := context.WithTimeout(context.Background(), time.Second*10)
	defer cancelF()
	req = req.WithContext(ctx)

	goapp.Log.Infof("Call: %s", goapp.Sanitize(req.URL.String()))
	resp, err := w.httpClient.Do(req)

	if err != nil {
		return errors.Wrapf(err, "can't call '%s'", req.URL.String())
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 10000))
		_ = resp.Body.Close()
	}()
	if err := goapp.ValidateHTTPResp(resp, 100); err != nil {
		return errors.Wrapf(err, "can't invoke '%s'", req.URL.String())
	}
	return nil
}
