package statusservice

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/pkg/errors"

	"github.com/airenas/big-tts/internal/pkg/persistence"
	"github.com/airenas/go-app/pkg/goapp"

	slog "log"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

// StatusProvider returns status for the ID
type StatusProvider interface {
	Get(ctx context.Context, id string) (*persistence.Status, error)
}

// Data keeps data required for service work
type Data struct {
	Port           int
	StatusProvider StatusProvider
}

// StartWebServer starts echo web service
func StartWebServer(ctx context.Context, data *Data) error {
	log.Ctx(ctx).Info().Msgf("Starting BIG TTS Status service at %d", data.Port)

	if data.StatusProvider == nil {
		return errors.New("no status provider")
	}

	portStr := strconv.Itoa(data.Port)

	e := initRoutes(ctx, data)

	e.Server.Addr = ":" + portStr
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 10 * time.Second
	e.Server.WriteTimeout = 10 * time.Second

	gracehttp.SetLogger(slog.New(goapp.Log, "", 0))

	return gracehttp.Serve(e.Server)
}

var promMdlw *prometheus.Prometheus

func init() {
	promMdlw = prometheus.NewPrometheus("tts_status", nil)
}

func initRoutes(ctx context.Context, data *Data) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	promMdlw.Use(e)

	e.GET("/status/:id", status(data))
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
		st, err := data.StatusProvider.Get(c.Request().Context(), id)
		if err != nil {
			log.Ctx(c.Request().Context()).Error().Err(err).Msg("Failed to get status")
			return echo.NewHTTPError(http.StatusInternalServerError, "Service error")
		}
		if st == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "No status by ID")
		}
		res := result{Status: st.Status, Error: st.Error}
		return c.JSON(http.StatusOK, res)
	}
}
