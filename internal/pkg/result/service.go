package result

import (
	"context"
	"net/http"
	"strconv"
	"time"

	slog "log"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/airenas/async-api/pkg/api"
	"github.com/airenas/go-app/pkg/goapp"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// FileReader loads file by name
type FileReader interface {
	Load(name string) (api.FileRead, error)
}

// FileNameProvider provides name for result file
type FileNameProvider interface {
	GetResultFile(id string) (string, error)
}

// Data keeps data required for service work
type Data struct {
	Port         int
	Reader       FileReader
	NameProvider FileNameProvider
}

// StartWebServer starts echo web service
func StartWebServer(ctx context.Context, data *Data) error {
	log.Ctx(ctx).Info().Msgf("Starting BIG TTS Result service at %d", data.Port)

	if err := validate(data); err != nil {
		return err
	}

	portStr := strconv.Itoa(data.Port)

	e := initRoutes(ctx, data)

	e.Server.Addr = ":" + portStr
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 10 * time.Second
	e.Server.WriteTimeout = 5 * time.Minute

	gracehttp.SetLogger(slog.New(goapp.Log, "", 0))

	return gracehttp.Serve(e.Server)
}

func validate(data *Data) error {
	if data.Reader == nil {
		return errors.New("no file reader")
	}
	if data.NameProvider == nil {
		return errors.New("no name provider")
	}
	return nil
}

var promMdlw *prometheus.Prometheus

func init() {
	promMdlw = prometheus.NewPrometheus("tts_result", nil)
}

func initRoutes(ctx context.Context, data *Data) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	promMdlw.Use(e)

	e.GET("/result/:id", download(data))
	e.HEAD("/result/:id", download(data))
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

func download(data *Data) func(echo.Context) error {
	return func(c echo.Context) error {
		defer goapp.Estimate("download method")()

		id := c.Param("id")
		if id == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "No ID")
		}
		fileName, err := data.NameProvider.GetResultFile(id)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "No file by ID")
		}

		ctx := c.Request().Context()

		file, err := data.Reader.Load(fileName)
		if err != nil {
			log.Ctx(ctx).Err(err).Send()
			return echo.NewHTTPError(http.StatusInternalServerError, "Can't get file")
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			log.Ctx(ctx).Err(err).Send()
			return echo.NewHTTPError(http.StatusInternalServerError, "Can't get file")
		}

		w := c.Response()
		w.Header().Set("Content-Disposition", "attachment; filename="+fileInfo.Name())
		http.ServeContent(w, c.Request(), fileInfo.Name(), fileInfo.ModTime(), file)
		return nil
	}
}
