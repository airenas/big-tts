package clean

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/pkg/errors"

	"github.com/airenas/go-app/pkg/goapp"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Cleaner interface {
	Clean(ID string) error
}

// Data keeps data required for service work
type Data struct {
	Port    int
	Cleaner Cleaner
}

//StartWebServer starts echo web service
func StartWebServer(data *Data) error {
	goapp.Log.Infof("Starting BIG TTS Clean service at %d", data.Port)

	if data.Cleaner == nil {
		return errors.New("no cleaner")
	}

	portStr := strconv.Itoa(data.Port)

	e := initRoutes(data)

	e.Server.Addr = ":" + portStr
	e.Server.Addr = ":" + portStr
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 5 * time.Second
	e.Server.WriteTimeout = 30 * time.Second

	w := goapp.Log.Writer()
	defer w.Close()
	l := log.New(w, "", 0)
	gracehttp.SetLogger(l)

	return gracehttp.Serve(e.Server)
}

var promMdlw *prometheus.Prometheus

func init() {
	promMdlw = prometheus.NewPrometheus("tts_clean", nil)
}

func initRoutes(data *Data) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	promMdlw.Use(e)

	e.DELETE("/delete/:id", delete(data.Cleaner))
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

func delete(cleaner Cleaner) func(echo.Context) error {
	return func(c echo.Context) error {
		defer goapp.Estimate("delete method")()

		id := c.Param("id")
		if id == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "No ID")
		}
		err := cleaner.Clean(id)
		if err != nil {
			goapp.Log.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Can't delete")
		}
		return c.String(http.StatusOK, "deleted")
	}
}
