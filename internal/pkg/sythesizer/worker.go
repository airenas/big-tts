package sythesizer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/airenas/big-tts/internal/pkg/utils"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
)

type Worker struct {
	inDir      string
	outDir     string
	serviceURL string

	loadFunc      func(string) ([]byte, error)
	saveFunc      func(string, []byte) error
	createDirFunc func(string) error
	existsFunc    func(string) bool
	callFunc      func(string) ([]byte, error)
}

func NewWorker(inTemplate, outTemplate string, url string) (*Worker, error) {
	if !strings.Contains(inTemplate, "{}") {
		return nil, errors.Errorf("no ID template in inTemplate")
	}
	if !strings.Contains(outTemplate, "{}") {
		return nil, errors.Errorf("no ID template in outTemplate")
	}
	if url == "" {
		return nil, errors.Errorf("no service URL")
	}
	res := &Worker{inDir: inTemplate, outDir: outTemplate, serviceURL: url}
	res.loadFunc = ioutil.ReadFile
	res.saveFunc = utils.WriteFile
	res.existsFunc = utils.FileExists
	res.createDirFunc = func(name string) error { return os.MkdirAll(name, os.ModePerm) }
	res.callFunc = res.invokeService
	return res, nil
}

func (w *Worker) Do(ID string) error {
	goapp.Log.Infof("Doing synthesize job for %s", ID)
	var err error
	outDir := strings.ReplaceAll(w.outDir, "{}", ID)
	if err := w.createDirFunc(outDir); err != nil {
		return errors.Wrapf(err, "can't create %s", outDir)
	}
	stop := false
	for i := 0; !stop; i++ {
		stop, err = w.processFile(i, ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) processFile(num int, ID string) (bool, error) {
	inFile := filepath.Join(strings.ReplaceAll(w.inDir, "{}", ID), fmt.Sprintf("%04d.txt", num))
	outDir := strings.ReplaceAll(w.outDir, "{}", ID)
	outFile := filepath.Join(outDir, fmt.Sprintf("%04d.mp3", num))
	if !w.existsFunc(inFile) {
		return true, nil
	}
	if w.existsFunc(outFile) {
		return false, nil
	}
	return false, w.invoke(inFile, outFile)
}

func (w *Worker) invoke(inFile string, outFile string) error {
	text, err := w.loadFunc(inFile)
	if err != nil {
		return err
	}
	bytes, err := w.callFunc(string(text))
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
	}

	result struct {
		AudioAsString string `json:"audioAsString,omitempty"`
		Error         string `json:"error,omitempty"`
	}
)

func (w *Worker) invokeService(data string) ([]byte, error) {
	inp := input{Text: data, OutputFormat: "mp3", Voice: "astra"}
	var out result
	err := invoke(w.serviceURL, inp, &out)
	if (err != nil) {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(out.AudioAsString)
}

func invoke(URL string, dataIn input, dataOut *result) error {
	b := new(bytes.Buffer)
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(dataIn)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", URL, b)
	if err != nil {
		return errors.Wrapf(err, "can't prepare request to '%s'", URL)
	}
	req.Header.Set("Content-Type", "application/json")

	ctx, cancelF := context.WithTimeout(context.Background(), time.Minute * 10)
	defer cancelF()
	req = req.WithContext(ctx)
	goapp.Log.Info("Call : ", req.URL.String())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "can't call '%s'", req.URL.String())
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.Errorf("can't invoke '%s'. Code: '%d'", req.URL.String(), resp.StatusCode)
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
