package upload

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/airenas/big-tts/internal/pkg/persistence"
	"github.com/airenas/go-app/pkg/goapp"
	"github.com/labstack/echo/v4"

	"github.com/pkg/errors"
)

const (
	headerDefaultFormat = "x-tts-default-output-format"
	headerCollectData   = "x-tts-collect-data"
	//HeaderSaveTags http header name for providing tags for saving to DB
	HeaderSaveTags      = "x-tts-save-tags"
)

//TTSConfigutaror tts request configuration
type TTSConfigutaror struct {
	defaultOutputFormat string
	defaultVoice        string
	availableVoices     map[string]bool
}

//NewTTSConfigurator creates the initial request configuration
func NewTTSConfigurator(format, voice string, voices []string) (*TTSConfigutaror, error) {
	res := &TTSConfigutaror{}
	var err error
	res.defaultOutputFormat, err = getOutputAudioFormat(format)
	if err != nil {
		return nil, errors.Wrap(err, "can't init default format")
	}
	goapp.Log.Infof("Default output format: %s", res.defaultOutputFormat)
	res.defaultVoice, res.availableVoices, err = initVoices(voice, voices)
	if err != nil {
		return nil, errors.Wrap(err, "can't init voices")
	}
	goapp.Log.Infof("Voices. Default: %s, all: %v", res.defaultVoice, res.availableVoices)
	return res, nil
}

//Configure prepares request configuration
func (c *TTSConfigutaror) Configure(e echo.Context) (*persistence.ReqData, error) {
	res := &persistence.ReqData{}
	var err error
	r := e.Request()
	res.OutputFormat, err = getOutputAudioFormat(defaultS(e.FormValue("outputFormat"), getHeader(r, headerDefaultFormat)))
	if err != nil {
		return nil, err
	}
	if res.OutputFormat == "" {
		res.OutputFormat = c.defaultOutputFormat
	}

	res.SaveRequest, err = getAllowCollect(getBool(e.FormValue("saveRequest")), getHeader(r, headerCollectData))
	if err != nil {
		return nil, err
	}
	res.SaveTags = getSaveTags(getHeader(r, HeaderSaveTags))

	res.Speed, err = getSpeed(e.FormValue("speed"))
	if err != nil {
		return nil, err
	}
	res.Voice, err = c.getVoice(e.FormValue("voice"))
	if err != nil {
		return nil, err
	}
	res.Email = e.FormValue("email")
	return res, nil
}

func getBool(s string) *bool {
	if s == "" {
		return nil
	}
	res := s == "true" || s == "1"
	return &res
}

func getAllowCollect(v *bool, s string) (bool, error) {
	st := strings.TrimSpace(strings.ToLower(s))
	if st == "" || st == "request" {
		return v != nil && *v, nil
	}
	if v == nil {
		return st == "always", nil
	}
	if st == "always" && *v {
		return true, nil
	}
	if st == "never" && !*v {
		return false, nil
	}
	return false, errors.Errorf("AllowCollectData=%t is rejected for this key.", *v)
}

func getOutputAudioFormat(s string) (string, error) {
	st := strings.TrimSpace(s)
	if st == "mp3" || st == "m4a" || st == "" {
		return st, nil
	}
	return "", errors.Errorf("unknown audio format '%s'", s)
}

func initVoices(def string, all []string) (string, map[string]bool, error) {
	resVoice := strings.TrimSpace(def)
	if resVoice == "" {
		return "", nil, errors.New("no default voice")
	}
	resAll := make(map[string]bool)
	resAll[resVoice] = true
	for _, s := range all {
		s = strings.TrimSpace(s)
		if s != "" {
			resAll[s] = true
		}
	}
	return resVoice, resAll, nil
}

func (c *TTSConfigutaror) getVoice(voice string) (string, error) {
	if voice == "" {
		return c.defaultVoice, nil
	}
	if c.availableVoices[voice] {
		return voice, nil
	}
	return "", errors.Errorf("unknown voice '%s'", voice)
}

func getHeader(r *http.Request, key string) string {
	return r.Header.Get(key)
}

func getSaveTags(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.Split(strings.TrimSpace(v), ",")
}

func getSpeed(vs string) (float64, error) {
	if vs == "" {
		return 1, nil
	}
	v, err := strconv.ParseFloat(vs, 64)
	if err != nil {
		return 0, errors.Errorf("wrong speed value %s.", vs)
	}
	if !(v < 0.000001 && v > -0.00001) {
		if v < 0.5 || v > 2.0 {
			return 0, errors.Errorf("speed value (%.2f) must be in [0.5,2].", v)
		}
	}
	return v, nil
}

func defaultS(s, s1 string) string {
	if strings.TrimSpace(s) != "" {
		return s
	}
	return s1
}
