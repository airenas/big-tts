package splitter

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/utils"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/airenas/tts-line/pkg/ssml"
	"github.com/pkg/errors"
)

// Worker for implementing text split
type Worker struct {
	loadPath string
	savePath string

	loadFunc      func(string) ([]byte, error)
	saveFunc      func(string, []byte) error
	createDirFunc func(string) error
	wantedChars   int
}

// NewWorker initiates new worker
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

// Do main worker's method
func (w *Worker) Do(ctx context.Context, msg *messages.TTSMessage) error {
	goapp.Log.Infof("Doing split job for %s", msg.ID)
	text, err := w.load(msg.ID)
	if err != nil {
		return errors.Wrapf(err, "can't load text")
	}
	texts, err := w.split(text, msg.Voice, msg.Speed)
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

func (w *Worker) split(text string, voice string, speed float64) ([]string, error) {
	if strings.HasPrefix(text, "<speak") {
		return w.doSSML(text, voice, speed)
	}
	return w.splitText(text)
}

func (w *Worker) doSSML(text string, voice string, speed float64) ([]string, error) {
	parts, err := ssml.Parse(strings.NewReader(text), &ssml.Text{Voice: voice, Speed: float32(speed)},
		func(s string) (string, error) { return s, nil })
	if err != nil {
		return nil, fmt.Errorf("can't parse: %v", err)
	}

	var res []string
	var cParts []ssml.Part
	cLen := 0
	for _, part := range parts {
		switch sp := part.(type) {
		case *ssml.Text:
			txts, err := w.splitText(sp.Text)
			if err != nil {
				return nil, errors.Wrapf(err, "can't split")
			}
			for _, txt := range txts {
				pLen := utf8.RuneCountInString(txt)
				if cLen+pLen > w.wantedChars {
					if len(cParts) > 0 {
						res = append(res, saveToSSMLString(cParts))
					}
					cParts, cLen = nil, 0
				}
				cParts, cLen = append(cParts, &ssml.Text{Text: txt, Voice: sp.Voice, Speed: sp.Speed}), cLen+pLen
			}
		case *ssml.Pause:
			cParts = append(cParts, sp)
		default:
			return nil, fmt.Errorf("unknown type %T", sp)
		}
	}
	if len(cParts) > 0 {
		res = append(res, saveToSSMLString(cParts))
	}
	return res, nil
}

func saveToSSMLString(cParts []ssml.Part) string {
	res := strings.Builder{}
	res.WriteString("<speak>")
	for _, part := range cParts {
		switch sp := part.(type) {
		case *ssml.Text:
			res.WriteString(fmt.Sprintf(`<voice name="%s"><prosody rate="%s">`, sp.Voice, toRateStr(sp.Speed)))
			_ = xml.EscapeText(&res, []byte(sp.Text))
			res.WriteString(`</prosody></voice>`)
		case *ssml.Pause:
			res.WriteString(fmt.Sprintf(`<break time="%dms"/>`, sp.Duration.Milliseconds()))
		default:
			panic(fmt.Errorf("unknown type %T", sp))
		}
	}
	res.WriteString("</speak>")
	return res.String()
}

func toRateStr(f float32) string {
	if f > 2 {
		f = 2
	} else if f < .5 {
		f = .5
	}
	p := 100
	if f > 1 {
		p = int(150 - 50*f)
	} else {
		p = int(300 - 200*f)
	}
	return fmt.Sprintf("%d%%", p)
}

func (w *Worker) splitText(text string) ([]string, error) {
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
