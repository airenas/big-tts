package upload

import (
	"context"
	"io"
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

	slog "log"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

// FileSaver provides save file functionality
type FileSaver interface {
	Save(name string, r io.Reader) error
}

// MsgSender provides send msg functionality
type MsgSender interface {
	Send(msg amessages.Message, queue, replyQueue string) error
}

// RequestSaver saves requests to DB
type RequestSaver interface {
	Save(ctx context.Context, req *persistence.ReqData) error
}

// Data keeps data required for service work
type Data struct {
	Port         int
	Configurator *TTSConfigutaror
	Saver        FileSaver
	ReqSaver     RequestSaver
	MsgSender    MsgSender
}

const requestIDHEader = "x-doorman-requestid"

// StartWebServer starts echo web service
func StartWebServer(ctx context.Context, data *Data) error {
	log.Ctx(ctx).Info().Msgf("Starting HTTP BIG TTS Line service at %d", data.Port)
	if err := validate(data); err != nil {
		return err
	}

	portStr := strconv.Itoa(data.Port)

	e := initRoutes(ctx, data)

	e.Server.Addr = ":" + portStr
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 45 * time.Second
	e.Server.WriteTimeout = 30 * time.Second

	gracehttp.SetLogger(slog.New(goapp.Log, "", 0))

	return gracehttp.Serve(e.Server)
}

func validate(data *Data) error {
	if data.Saver == nil {
		return errors.New("no file saver")
	}
	if data.ReqSaver == nil {
		return errors.New("no request saver")
	}
	if data.Configurator == nil {
		return errors.New("no configurator")
	}
	if data.MsgSender == nil {
		return errors.New("no msg sender")
	}
	return nil
}

var promMdlw *prometheus.Prometheus

func init() {
	promMdlw = prometheus.NewPrometheus("tts_upload", nil)
}

func initRoutes(ctx context.Context, data *Data) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	promMdlw.Use(e)

	e.POST("/upload", upload(data))
	e.GET("/live", live(data))

	log.Ctx(ctx).Info().Msg("Routes:")
	for _, r := range e.Routes() {
		log.Ctx(ctx).Info().Msgf("  %s %s", r.Method, r.Path)
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

		ctx := c.Request().Context()

		inData, err := getInputData(c, data.Configurator)
		if err != nil {
			log.Ctx(ctx).Err(err).Send()
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		form, err := c.MultipartForm()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "no multipart form data")
		}
		defer cleanFiles(context.Background(), form)

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
			log.Ctx(ctx).Err(err).Send()
			return errors.Wrap(err, "can not save file")
		}

		requestID := extractRequestID(c.Request().Header)
		log.Ctx(ctx).Info().Msgf("RequestID=%s", goapp.Sanitize(requestID))

		inData.ID = id
		inData.Filename = fileName
		inData.RequestID = requestID
		err = data.ReqSaver.Save(ctx, inData)
		if err != nil {
			log.Ctx(ctx).Err(err).Send()
			return errors.Wrap(err, "can not save request")
		}

		msg := &messages.TTSMessage{
			QueueMessage: amessages.QueueMessage{ID: id},
			Voice:        inData.Voice,
			SaveRequest:  inData.SaveRequest,
			Speed:        inData.Speed,
			OutputFormat: inData.OutputFormat,
			SaveTags:     inData.SaveTags,
			RequestID:    requestID,
		}
		err = data.MsgSender.Send(msg, messages.Upload, "")
		if err != nil {
			log.Ctx(ctx).Err(err).Send()
			return errors.Wrap(err, "can not send msg")
		}

		res := result{ID: id}
		return c.JSON(http.StatusOK, res)
	}
}

func extractRequestID(header http.Header) string {
	return header.Get(requestIDHEader)
}

func getInputData(c echo.Context, cfg *TTSConfigutaror) (*persistence.ReqData, error) {
	return cfg.Configure(c)
}

func cleanFiles(ctx context.Context, f *multipart.Form) {
	if f != nil {
		if err := f.RemoveAll(); err != nil {
			log.Ctx(ctx).Err(err).Send()
		}
	}
}

func checkFileExtension(ext string) bool {
	return ext == ".txt"
}
