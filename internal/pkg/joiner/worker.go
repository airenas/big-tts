package joiner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/airenas/big-tts/internal/pkg/utils"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
)

type Worker struct {
	inDir    string
	savePath string

	existsFunc    func(string) bool
	saveFunc      func(string, []byte) error
	createDirFunc func(string) error
}

func NewWorker(inDir string, savePath string) (*Worker, error) {
	if !strings.Contains(inDir, "{}") {
		return nil, errors.Errorf("no ID template in inDir")
	}
	if !strings.Contains(savePath, "{}") {
		return nil, errors.Errorf("no ID template in savePath")
	}
	goapp.Log.Infof("Joiner in: %s", inDir)
	goapp.Log.Infof("Joiner out: %s", savePath)
	res := &Worker{inDir: inDir, savePath: savePath}
	res.existsFunc = utils.FileExists
	res.saveFunc = utils.WriteFile
	res.createDirFunc = func(name string) error { return os.MkdirAll(name, os.ModePerm) }
	return res, nil
}

func (w *Worker) Do(ID string) error {
	goapp.Log.Infof("Doing join job for %s", ID)
	files, err := w.makeList(ID)
	if err != nil {
		return errors.Wrapf(err, "can't prepare files list")
	}
	path := strings.ReplaceAll(w.savePath, "{}", ID)
	err = w.createDirFunc(path)
	if err != nil {
		return errors.Wrapf(err, "can't create %s", path)
	}
	listFile := filepath.Join(path, "list.txt")
	err = w.saveFunc(listFile, []byte(prepareListFile(files)))
	if err != nil {
		return errors.Wrapf(err, "can't save %s", listFile)
	}
	return nil
}

func (w *Worker) makeList(ID string) ([]string, error) {
	path := strings.ReplaceAll(w.inDir, "{}", ID)
	var res []string

	for i := 0; ; i++ {
		inFile := filepath.Join(path, fmt.Sprintf("%04d.mp3", i))
		if w.existsFunc(inFile) {
			res = append(res, inFile)
		} else {
			break
		}
	}
	return res, nil
}

func prepareListFile(files [] string) (string) {
	res := strings.Builder{}

	for _, s := range files {
		res.WriteString(fmt.Sprintf("file '%s'\n", s))
	}
	return res.String()
}
