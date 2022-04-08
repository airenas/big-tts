package statusservice

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/pkg/errors"

	"github.com/airenas/big-tts/internal/pkg/persistence"
	"github.com/airenas/go-app/pkg/goapp"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// StatusProvider returns status for the ID
type StatusProvider interface {
	Get(id string) (*persistence.Status, error)
}

// Data keeps data required for service work
type Data struct {
	Port           int
	StatusProvider StatusProvider
}

//StartWebServer starts echo web service
func StartWebServer(data *Data) error {
	goapp.Log.Infof("Starting BIG TTS Status service at %d", data.Port)

	if data.StatusProvider == nil {
		return errors.New("no status provider")
	}

	portStr := strconv.Itoa(data.Port)

	e := initRoutes(data)

	e.Server.Addr = ":" + portStr
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 10 * time.Second
	e.Server.WriteTimeout = 10 * time.Second

	w := goapp.Log.Writer()
	defer w.Close()
	l := log.New(w, "", 0)
	gracehttp.SetLogger(l)

	return gracehttp.Serve(e.Server)
}

var promMdlw *prometheus.Prometheus

func init() {
	promMdlw = prometheus.NewPrometheus("tts_status", nil)
}

func initRoutes(data *Data) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	promMdlw.Use(e)

	e.GET("/status/:id", status(data))
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
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func status(data *Data) func(echo.Context) error {
	return func(c echo.Context) error {
		defer goapp.Estimate("status method")()

		id := c.Param("id")
		if id == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "No ID")
		}
		st, err := data.StatusProvider.Get(id)
		if err != nil {
			goapp.Log.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Service error")
		}
		if st == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "No status by ID")
		}
		res := result{Status: st.Status, Error: st.Error}
		return c.JSON(http.StatusOK, res)
	}
}
