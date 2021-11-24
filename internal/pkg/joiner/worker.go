package joiner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/utils"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/pkg/errors"
)

type Worker struct {
	inDir    string
	savePath string
	workPath string
	metadata []string

	existsFunc    func(string) bool
	saveFunc      func(string, []byte) error
	createDirFunc func(string) error
	convertFunc   func([]string) error
}

func NewWorker(inDir, savePath, workPath string, metadata []string) (*Worker, error) {
	if !strings.Contains(inDir, "{}") {
		return nil, errors.Errorf("no ID template in inDir")
	}
	if !strings.Contains(savePath, "{}") {
		return nil, errors.Errorf("no ID template in savePath")
	}
	if !strings.Contains(workPath, "{}") {
		return nil, errors.Errorf("no ID template in workPath")
	}
	goapp.Log.Infof("Joiner in: %s", inDir)
	goapp.Log.Infof("Joiner out: %s", savePath)
	res := &Worker{inDir: inDir, savePath: savePath, metadata: metadata, workPath: workPath}
	res.existsFunc = utils.FileExists
	res.saveFunc = utils.WriteFile
	res.convertFunc = runCmd
	res.createDirFunc = func(name string) error { return os.MkdirAll(name, os.ModePerm) }
	return res, nil
}

func (w *Worker) Do(msg *messages.TTSMessage) error {
	goapp.Log.Infof("Doing join job for %s", msg.ID)
	files, err := w.makeList(msg.ID, msg.OutputFormat)
	if err != nil {
		return errors.Wrapf(err, "can't prepare files list")
	}
	path := strings.ReplaceAll(w.savePath, "{}", msg.ID)
	err = w.createDirFunc(path)
	if err != nil {
		return errors.Wrapf(err, "can't create %s", path)
	}
	wpath := strings.ReplaceAll(w.workPath, "{}", msg.ID)
	err = w.createDirFunc(wpath)
	if err != nil {
		return errors.Wrapf(err, "can't create %s", wpath)
	}
	listFile := filepath.Join(wpath, "list.txt")
	err = w.saveFunc(listFile, []byte(prepareListFile(files)))
	if err != nil {
		return errors.Wrapf(err, "can't save %s", listFile)
	}
	outFile := filepath.Join(path, fmt.Sprintf("result.%s", msg.OutputFormat))
	return w.join(listFile, outFile)
}

func (w *Worker) makeList(ID, format string) ([]string, error) {
	path := strings.ReplaceAll(w.inDir, "{}", ID)
	var res []string

	for i := 0; ; i++ {
		inFile := filepath.Join(path, fmt.Sprintf("%04d.%s", i, format))
		if w.existsFunc(inFile) {
			res = append(res, inFile)
		} else {
			break
		}
	}
	return res, nil
}

func prepareListFile(files []string) string {
	res := strings.Builder{}

	for _, s := range files {
		res.WriteString(fmt.Sprintf("file '%s'\n", s))
	}
	return res.String()
}

func (w *Worker) join(nameIn string, out string) error {
	params := []string{"ffmpeg", "-f", "concat", "-safe", "0", "-i", nameIn, "-c", "copy"}
	params = append(params, getMetadataParams(w.metadata)...)
	params = append(params, out)
	err := w.convertFunc(params)
	if err != nil {
		return err
	}
	return nil
}

func getMetadataParams(prm []string) []string {
	res := []string{}
	for _, p := range prm {
		pt := strings.TrimSpace(p)
		if pt != "" {
			res = append(res, "-metadata")
			res = append(res, pt)
		}
	}
	return res
}

func runCmd(cmdArr []string) error {
	goapp.Log.Infof("Run: %s", strings.Join(cmdArr, " "))
	cmd := exec.Command(cmdArr[0], cmdArr[1:]...)
	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "Output: "+outputBuffer.String())
	}
	return nil
}
