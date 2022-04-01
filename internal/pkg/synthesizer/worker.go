package synthesizer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/upload"
	"github.com/airenas/big-tts/internal/pkg/utils"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
)

//Worker implements synthesize one part functionality
type Worker struct {
	inDir       string
	outDir      string
	serviceURL  string
	workerCount int
	httpClient  http.Client

	loadFunc      func(string) ([]byte, error)
	saveFunc      func(string, []byte) error
	createDirFunc func(string) error
	existsFunc    func(string) bool
	callFunc      func(string, *messages.TTSMessage) ([]byte, error)
}

//NewWorker creates new synthesize worker
func NewWorker(inTemplate, outTemplate string, url string, workerCount int) (*Worker, error) {
	if !strings.Contains(inTemplate, "{}") {
		return nil, errors.Errorf("no ID template in inTemplate")
	}
	if !strings.Contains(outTemplate, "{}") {
		return nil, errors.Errorf("no ID template in outTemplate")
	}
	if url == "" {
		return nil, errors.Errorf("no service URL")
	}
	if workerCount < 1 {
		return nil, errors.Errorf("no workerCount provided")
	}
	res := &Worker{inDir: inTemplate, outDir: outTemplate, serviceURL: url}
	res.loadFunc = ioutil.ReadFile
	res.saveFunc = utils.WriteFile
	res.existsFunc = utils.FileExists
	res.createDirFunc = func(name string) error { return os.MkdirAll(name, os.ModePerm) }
	res.callFunc = res.invokeService
	res.workerCount = workerCount
	res.httpClient = http.Client{Transport: &http.Transport{
		MaxIdleConns:        40,
		MaxIdleConnsPerHost: 40,
		IdleConnTimeout:     90 * time.Second,
		MaxConnsPerHost:     50,
	}}

	goapp.Log.Infof("Synthesizer URL: %s", res.serviceURL)
	goapp.Log.Infof("Synthesizer workers: %d", res.workerCount)
	goapp.Log.Infof("Synthesizer in dir: %s", res.inDir)
	goapp.Log.Infof("Synthesizer out dir: %s", res.outDir)
	return res, nil
}

//Do synthesizes one part of a text
func (w *Worker) Do(ctx context.Context, msg *messages.TTSMessage) error {
	goapp.Log.Infof("Doing synthesize job for %s", msg.ID)
	outDir := strings.ReplaceAll(w.outDir, "{}", msg.ID)
	if err := w.createDirFunc(outDir); err != nil {
		return errors.Wrapf(err, "can't create %s", outDir)
	}

	errCh := make(chan error, w.workerCount+1)
	syncCh := make(chan struct{}, w.workerCount)
	stop := false
	wg := &sync.WaitGroup{}
	var inF, outF string
out:
	for i := 0; !stop; i++ {
		stop, inF, outF = w.getFiles(i, msg)
		if inF != "" {
			// make sure we exit in case of error or cancelling before
			// --- case syncCh <- struct{}{}: ---
			select {
			case <-ctx.Done():
				goapp.Log.Warnf("Exit synthesizer loop")
				errCh <- context.Canceled
				break out
			case err := <-errCh:
				goapp.Log.Infof("Error occured, waiting to complete all jobs")
				wg.Wait()
				return err
			default:
			}

			select {
			case syncCh <- struct{}{}:
			case <-ctx.Done():
				goapp.Log.Warnf("Exit synthesizer loop")
				errCh <- context.Canceled
				break out
			case err := <-errCh:
				goapp.Log.Infof("Error occured, waiting to complete all jobs")
				wg.Wait()
				return err
			}
			wg.Add(1)
			go func(_inF, _outF string, _i int) {
				defer func() {
					wg.Done()
					<-syncCh
				}()
				goapp.Log.Infof("Process item %d", _i)
				err := w.invoke(_inF, _outF, msg)
				if err != nil {
					errCh <- err
				}
			}(inF, outF, i)
		}
	}
	goapp.Log.Infof("Waiting to complete all jobs")
	wg.Wait()
	errCh <- nil
	return <-errCh
}

func (w *Worker) getFiles(num int, msg *messages.TTSMessage) (bool, string, string) {
	inFile := filepath.Join(strings.ReplaceAll(w.inDir, "{}", msg.ID), fmt.Sprintf("%04d.txt", num))
	if !w.existsFunc(inFile) {
		return true, "", ""
	}
	outDir := strings.ReplaceAll(w.outDir, "{}", msg.ID)
	outFile := filepath.Join(outDir, fmt.Sprintf("%04d.%s", num, msg.OutputFormat))
	if w.existsFunc(outFile) {
		return false, "", ""
	}
	return false, inFile, outFile
}

func (w *Worker) invoke(inFile string, outFile string, msg *messages.TTSMessage) error {
	text, err := w.loadFunc(inFile)
	if err != nil {
		return err
	}
	bytes, err := w.callFunc(string(text), msg)
	if err != nil {
		return err
	}
	return w.saveFunc(outFile, bytes)
}

type (
	input struct {
		Text string `json:"text,omitempty"`
		//Possible values are m4a, mp3
		OutputFormat     string  `json:"outputFormat,omitempty"`
		OutputTextFormat string  `json:"outputTextFormat,omitempty"`
		AllowCollectData *bool   `json:"saveRequest,omitempty"`
		Speed            float32 `json:"speed,omitempty"`
		Voice            string  `json:"voice,omitempty"`
		Priority         int     `json:"priority,omitempty"`
	}

	result struct {
		AudioAsString string `json:"audioAsString,omitempty"`
		Error         string `json:"error,omitempty"`
	}
)

func (w *Worker) invokeService(data string, msg *messages.TTSMessage) ([]byte, error) {
	inp := input{Text: data, OutputFormat: msg.OutputFormat,
		Voice:            msg.Voice,
		Speed:            float32(msg.Speed),
		AllowCollectData: &msg.SaveRequest,
		Priority:         300} // will indicate 300s wait on high load comparing to priority=0
	var out result
	err := w.invokeRemote(inp, &out, msg.SaveTags)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(out.AudioAsString)
}

func (w *Worker) invokeRemote(dataIn input, dataOut *result, saveTags []string) error {
	b := new(bytes.Buffer)
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(dataIn)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", w.serviceURL, b)
	if err != nil {
		return errors.Wrapf(err, "can't prepare request to '%s'", w.serviceURL)
	}
	req.Header.Set("Content-Type", "application/json")
	if len(saveTags) > 0 {
		req.Header.Set(upload.HeaderSaveTags, strings.Join(saveTags, ","))
	}

	ctx, cancelF := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancelF()
	req = req.WithContext(ctx)
	goapp.Log.Info("Call: ", req.URL.String())
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "can't call '%s'", req.URL.String())
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 10000))
		_ = resp.Body.Close()
	}()
	if err := goapp.ValidateHTTPResp(resp, 100); err != nil {
		err = errors.Wrapf(err, "can't invoke '%s'", req.URL.String())
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return utils.NewErrNonRestorableUsage(err)
		}
		return err
	}
	br, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "can't read body")
	}
	err = json.Unmarshal(br, dataOut)
	if err != nil {
		return errors.Wrap(err, "can't decode response")
	}
	return nil
}
