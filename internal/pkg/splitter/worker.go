package splitter

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/airenas/big-tts/internal/pkg/messages"
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
	wantedChars   int
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
	res.wantedChars = 1900
	return res, nil
}

func (w *Worker) Do(ctx context.Context, msg *messages.TTSMessage) error {
	goapp.Log.Infof("Doing split job for %s", msg.ID)
	text, err := w.load(msg.ID)
	if err != nil {
		return errors.Wrapf(err, "can't load text")
	}
	texts, err := w.split(text)
	if err != nil {
		return errors.Wrapf(err, "can't split text")
	}
	err = w.save(msg.ID, texts)
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
	rns := []rune(text)
	for len(rns) > 0 {
		pos, err := getNextSplit(rns, w.wantedChars, w.wantedChars/4)
		if err != nil {
			return nil, err
		}
		res = append(res, string(rns[:pos]))
		rns = rns[pos:]
	}
	return res, nil
}

func getNextSplit(rns []rune, start, interval int) (int, error) {
	l := len(rns)
	if l < (start + interval) {
		return l, nil
	}

	s := "   "
	best := -1
	bestV := 0
	for i := 0; i < interval; i++ {
		r := rns[start+i]
		s = getNewPattern(s, r)
		if s == ".\n\n" || s == "\n\n\n" {
			return start + i, nil
		} else if s == ".\nU" {
			best = start + i - 1
			bestV = 3
		} else if (s == ". U") && bestV < 2 {
			best = start + i - 1
			bestV = 2
		} else if (r == ' ') && bestV < 1 {
			best = start + i
			bestV = 1
		}
	}
	if best > 0 {
		return best, nil
	}
	return 0, errors.New("no split position found")
}

func getNewPattern(str string, r rune) string {
	if r == '\n' && str[1] == '.' && str[2] == ' ' {
		return str[:2] + "\n"
	}
	if r == '\n' {
		return str[1:] + "\n"
	}
	if r == '.' {
		return str[1:] + "."
	}
	if unicode.IsSpace(r) || r == '\t' {
		if str[2] == ' ' || str[2] == '\n' {
			return str
		}
		return str[1:] + " "
	}
	if unicode.IsUpper(r) {
		return str[1:] + "U"
	}
	return str[1:] + "-"
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
