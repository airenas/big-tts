package upload

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	amessages "github.com/airenas/async-api/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/messages"
	"github.com/airenas/big-tts/internal/pkg/persistence"

	"github.com/airenas/go-app/pkg/goapp"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type FileSaver interface {
	Save(name string, r io.Reader) error
}

type MsgSender interface {
	Send(msg amessages.Message, queue, replyQueue string) error
}

type RequestSaver interface {
	Save(req *persistence.ReqData) error
}

// Data keeps data required for service work
type Data struct {
	Port         int
	Configurator *TTSConfigutaror
	Saver        FileSaver
	ReqSaver     RequestSaver
	MsgSender    MsgSender
}

//StartWebServer starts echo web service
func StartWebServer(data *Data) error {
	goapp.Log.Infof("Starting HTTP BIG TTS Line service at %d", data.Port)

	if data.Saver == nil {
		return errors.New("no file saver")
	}

	if data.ReqSaver == nil {
		return errors.New("no request saver")
	}

	portStr := strconv.Itoa(data.Port)

	e := initRoutes(data)

	e.Server.Addr = ":" + portStr
	e.Server.Addr = ":" + portStr
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 45 * time.Second
	e.Server.WriteTimeout = 30 * time.Second

	w := goapp.Log.Writer()
	defer w.Close()
	l := log.New(w, "", 0)
	gracehttp.SetLogger(l)

	return gracehttp.Serve(e.Server)
}

var promMdlw *prometheus.Prometheus

func init() {
	promMdlw = prometheus.NewPrometheus("tts", nil)
}

func initRoutes(data *Data) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	promMdlw.Use(e)

	e.POST("/upload", upload(data))
	e.GET("/live", live(data))

	goapp.Log.Info("Routes:")
	for _, r := range e.Routes() {
		goapp.Log.Infof("  %s %s", r.Method, r.Path)
	}
	return e
}

func live(data *Data) func(echo.Context) error {
	return func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, []byte(`{"service":"OK"}`))
	}
}

type result struct {
	ID string `json:"id"`
}

func upload(data *Data) func(echo.Context) error {
	return func(c echo.Context) error {
		defer goapp.Estimate("upload method")()

		inData, err := getInputData(c, data.Configurator)
		if err != nil {
			goapp.Log.Error(err)
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		form, err := c.MultipartForm()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "no multipart form data")
		}
		defer cleanFiles(form)

		files, ok := form.File["file"]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "no file")
		}
		if len(files) > 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "multiple files")
		}

		file := files[0]
		ext := filepath.Ext(file.Filename)
		ext = strings.ToLower(ext)
		if !checkFileExtension(ext) {
			return echo.NewHTTPError(http.StatusBadRequest, "wrong file type: "+ext)
		}

		id := uuid.New().String()
		fileName := id + ext

		src, err := file.Open()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "can't read file")
		}
		defer src.Close()

		err = data.Saver.Save(fileName, src)
		if err != nil {
			goapp.Log.Error(err)
			return errors.Wrap(err, "can not save file")
		}

		inData.ID = id
		inData.Filename = fileName
		err = data.ReqSaver.Save(inData)
		if err != nil {
			goapp.Log.Error(err)
			return errors.Wrap(err, "can not save request")
		}

		msg := &amessages.QueueMessage{ID: id}
		msg.Tags = append(msg.Tags, newTag(messages.Voice, inData.Voice))
		msg.Tags = append(msg.Tags, newTag(messages.Speed, fmt.Sprintf("%.2f", inData.Speed)))
		msg.Tags = append(msg.Tags, newTag(messages.Format, inData.OutputFormat))
		msg.Tags = append(msg.Tags, newTag(messages.SaveRequest, fmt.Sprintf("%v", inData.SaveRequest)))
		msg.Tags = append(msg.Tags, newTag(messages.SaveTags, strings.Join(inData.SaveTags, ",")))
		err = data.MsgSender.Send(msg, messages.Upload, "")
		if err != nil {
			goapp.Log.Error(err)
			return errors.Wrap(err, "can not send msg")
		}

		res := result{ID: id}
		return c.JSON(http.StatusOK, res)
	}
}

func newTag(k messages.TagsType, v string) amessages.Tag {
	return amessages.Tag{Key: k.Name(), Value: v}
}

func getInputData(c echo.Context, cfg *TTSConfigutaror) (*persistence.ReqData, error) {
	return cfg.Configure(c)
}

func cleanFiles(f *multipart.Form) {
	if f != nil {
		f.RemoveAll()
	}
}

func checkFileExtension(ext string) bool {
	return ext == ".txt"
}
