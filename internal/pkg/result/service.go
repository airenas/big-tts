package result

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/pkg/errors"

	"github.com/airenas/async-api/pkg/api"
	"github.com/airenas/go-app/pkg/goapp"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type FileReader interface {
	Load(name string) (api.FileRead, error)
}

type FileNameProvider interface {
	GetResultFile(id string) (string, error)
}

// Data keeps data required for service work
type Data struct {
	Port         int
	Reader       FileReader
	NameProvider FileNameProvider
}

//StartWebServer starts echo web service
func StartWebServer(data *Data) error {
	goapp.Log.Infof("Starting BIG TTS Result service at %d", data.Port)

	if data.Reader == nil {
		return errors.New("no file reader")
	}

	portStr := strconv.Itoa(data.Port)

	e := initRoutes(data)

	e.Server.Addr = ":" + portStr
	e.Server.Addr = ":" + portStr
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 10 * time.Second
	e.Server.WriteTimeout = 5 * time.Minute

	w := goapp.Log.Writer()
	defer w.Close()
	l := log.New(w, "", 0)
	gracehttp.SetLogger(l)

	return gracehttp.Serve(e.Server)
}

var promMdlw *prometheus.Prometheus

func init() {
	promMdlw = prometheus.NewPrometheus("tts_result", nil)
}

func initRoutes(data *Data) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	promMdlw.Use(e)

	e.GET("/result/:id", download(data))
	e.HEAD("/result/:id", download(data))
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
		file, err := data.Reader.Load(fileName)
		if err != nil {
			goapp.Log.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Can't get file")
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			goapp.Log.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Can't get file")
		}

		w := c.Response()
		w.Header().Set("Content-Disposition", "attachment; filename="+fileInfo.Name())
		http.ServeContent(w, c.Request(), fileInfo.Name(), fileInfo.ModTime(), file)
		return nil
	}
}
