package splitter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/airenas/big-tts/internal/pkg/utils"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
)

type Worker struct {
	loadPath string
	savePath string

	loadFunc      func(string) ([]byte, error)
	saveFunc      func(string, []byte) error
	createDirFunc func(string) error
}

func NewWorker(loadPath string, savePath string) (*Worker, error) {
	if !strings.Contains(loadPath, "{}") {
		return nil, errors.Errorf("no ID template in load path")
	}
	if !strings.Contains(savePath, "{}") {
		return nil, errors.Errorf("no ID template in save path")
	}
	res := &Worker{loadPath: loadPath, savePath: savePath}
	res.loadFunc = ioutil.ReadFile
	res.saveFunc = utils.WriteFile
	res.createDirFunc = func(name string) error { return os.MkdirAll(name, os.ModePerm) }
	return res, nil
}

func (w *Worker) Do(ID string) error {
	goapp.Log.Infof("Doing split job for %s", ID)
	text, err := w.load(ID)
	if err != nil {
		return errors.Wrapf(err, "can't load text")
	}
	texts, err := w.split(text)
	if err != nil {
		return errors.Wrapf(err, "can't split text")
	}
	err = w.save(ID, texts)
	if err != nil {
		return errors.Wrapf(err, "can't save texts")
	}
	return nil
}

func (w *Worker) load(ID string) (string, error) {
	path := strings.ReplaceAll(w.loadPath, "{}", ID)
	bytes, err := w.loadFunc(path)
	if err != nil {
		return "", errors.Wrapf(err, "can't load %s", path)
	}
	return string(bytes), nil
}

func (w *Worker) split(text string) ([]string, error) {
	var res []string
	for _, s := range strings.Split(text, "\n") {
		st := strings.TrimSpace(s)
		if (st != ""){
			res = append(res, st)
		}
	}
	return res, nil
}

func (w *Worker) save(ID string, texts []string) error {
	path := strings.ReplaceAll(w.savePath, "{}", ID)
	err := w.createDirFunc(path)
	if err != nil {
		return errors.Wrapf(err, "can't create %s", path)
	}
	for i, s := range texts {
		fp := filepath.Join(path, fmt.Sprintf("%04d.txt", i))
		err := w.saveFunc(fp, []byte(s))
		if err != nil {
			return errors.Wrapf(err, "can't save %s", fp)
		}
	}
	return nil
}


