package synthesizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
)

// Worker implements synthesize one part functionality
type Worker struct {
	inDir       string
	outDir      string
	serviceURL  string
	workerCount int
	httpClient  http.Client

	loadFunc      func(string) ([]byte, error)
	saveFunc      func(context.Context, string, []byte) error
	createDirFunc func(string) error
	existsFunc    func(string) bool
	callFunc      func(context.Context, string, *messages.TTSMessage) ([]byte, error)
}

// NewWorker creates new synthesize worker
func NewWorker(ctx context.Context, inTemplate, outTemplate string, url string, workerCount int) (*Worker, error) {
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
	res.loadFunc = os.ReadFile
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

	log.Ctx(ctx).Info().Msgf("Synthesizer URL: %s", res.serviceURL)
	log.Ctx(ctx).Info().Msgf("Synthesizer workers: %d", res.workerCount)
	log.Ctx(ctx).Info().Msgf("Synthesizer in dir: %s", res.inDir)
	log.Ctx(ctx).Info().Msgf("Synthesizer out dir: %s", res.outDir)
	return res, nil
}

// Do synthesizes one part of a text
func (w *Worker) Do(ctx context.Context, msg *messages.TTSMessage) error {
	log.Ctx(ctx).Info().Str("ID", msg.ID).Msg("synthesize job")
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
				log.Ctx(ctx).Warn().Msg("Exit synthesizer loop")
				errCh <- context.Canceled
				break out
			case err := <-errCh:
				log.Ctx(ctx).Info().Msg("Error occurred, waiting to complete all jobs")
				wg.Wait()
				return err
			default:
			}

			select {
			case syncCh <- struct{}{}:
			case <-ctx.Done():
				log.Ctx(ctx).Warn().Msg("Exit synthesizer loop")
				errCh <- context.Canceled
				break out
			case err := <-errCh:
				log.Ctx(ctx).Info().Msg("Error occurred, waiting to complete all jobs")
				wg.Wait()
				return err
			}
			wg.Add(1)
			go func(_inF, _outF string, _i int) {
				defer func() {
					wg.Done()
					<-syncCh
				}()
				log.Ctx(ctx).Info().Int("i", i).Msg("Process item")
				err := w.invoke(ctx, _inF, _outF, msg)
				if err != nil {
					errCh <- err
				}
			}(inF, outF, i)
		}
	}
	log.Ctx(ctx).Info().Msg("Waiting to complete all jobs")
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

func (w *Worker) invoke(ctx context.Context, inFile string, outFile string, msg *messages.TTSMessage) error {
	text, err := w.loadFunc(inFile)
	if err != nil {
		return err
	}
	bytes, err := w.callFunc(ctx, string(text), msg)
	if err != nil {
		return err
	}
	return w.saveFunc(ctx, outFile, bytes)
}

type (
	input struct {
		Text             string  `json:"text,omitempty"`
		OutputFormat     string  `json:"outputFormat,omitempty"`
		OutputTextFormat string  `json:"outputTextFormat,omitempty"`
		AllowCollectData *bool   `json:"saveRequest,omitempty"`
		Speed            float32 `json:"speed,omitempty"`
		Voice            string  `json:"voice,omitempty"`
		Priority         int     `json:"priority,omitempty"`
	}

	result struct {
		Audio     []byte `json:"audio,omitempty" msgpack:"audio,omitempty"`
		Error     string `json:"error,omitempty" msgpack:"error,omitempty"`
		Text      string `json:"text,omitempty" msgpack:"text,omitempty"`
		RequestID string `json:"requestID,omitempty" msgpack:"requestID,omitempty"`
	}
)

func (w *Worker) invokeService(ctx context.Context, data string, msg *messages.TTSMessage) ([]byte, error) {
	inp := input{Text: data, OutputFormat: msg.OutputFormat,
		Voice:            msg.Voice,
		Speed:            float32(msg.Speed),
		AllowCollectData: &msg.SaveRequest,
		Priority:         300} // will indicate 300s wait on high load comparing to priority=0
	var out result
	err := w.invokeRemote(ctx, inp, &out, msg.SaveTags)
	if err != nil {
		return nil, err
	}
	return out.Audio, nil
}

func (w *Worker) invokeRemote(ctx context.Context, dataIn input, dataOut *result, saveTags []string) error {
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
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationMsgpack)
	if len(saveTags) > 0 {
		req.Header.Set(upload.HeaderSaveTags, strings.Join(saveTags, ","))
	}

	ctx, cancelF := context.WithTimeout(ctx, time.Minute*10)
	defer cancelF()
	req = req.WithContext(ctx)
	log.Ctx(ctx).Info().Str("url", goapp.Sanitize(req.URL.String())).Msg("Call")
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
		if isNonRestorableErrCode(resp.StatusCode) {
			return utils.NewErrNonRestorableUsage(err)
		}
		return err
	}

	return w.unmarshalResponse(ctx, resp, dataOut)
}

func (w *Worker) unmarshalResponse(ctx context.Context, resp *http.Response, dataOut *result) error {
	contentType := resp.Header.Get(echo.HeaderContentType)
	if strings.Contains(contentType, echo.MIMEApplicationMsgpack) {
		msgpackDecoder := msgpack.NewDecoder(resp.Body)
		if err := msgpackDecoder.Decode(dataOut); err != nil {
			return errors.Wrap(err, "can't decode msgpack response")
		}
		return nil
	}
	return fmt.Errorf("wanted msgpack response: got '%s'", contentType)
}

func isNonRestorableErrCode(c int) bool {
	return c < 400 // restore all 4xx and 5xx errors
}
