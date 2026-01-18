package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/ericfialkowski/shorturl/dao"
	"github.com/ericfialkowski/shorturl/env"
	"github.com/ericfialkowski/shorturl/status"
	"github.com/ericfialkowski/shorturl/telemetry"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	appPath     string = "/:abv"
	statsPath   string = "/:abv/stats"
	statsUiPath string = "/:abv/stats/ui"
	metricsPath string = "/diag/metrics"
	statusPath  string = "/diag/status"
)

type (
	Handlers struct {
		dao         dao.ShortUrlDao
		metrics     metrics
		otelMetrics *telemetry.Metrics
		startTime   time.Time
		status      *status.SimpleStatus
		id          string
	}

	metrics struct {
		Redirects uint64 `json:"redirect_counts"`
		UrlStats  uint64 `json:"redirect_stats_counts"`
		NewUrls   uint64 `json:"new_url_counts"`
		Deletes   uint64 `json:"delete_counts"`
		Metrics   uint64 `json:"metric_request_counts"`
		Status    uint64 `json:"stats_requests_counts"`
		Uptime    string `json:"uptime"`
	}

	urlReturn struct {
		Abv         string `json:"abv"`
		UrlLink     string `json:"url_link"`
		StatsLink   string `json:"stats_link"`
		StatsUiLink string `json:"stats_ui_link"`
	}
)

func createReturn(abv string) urlReturn {
	return urlReturn{
		Abv:         abv,
		UrlLink:     fmt.Sprintf("/%s", abv),
		StatsLink:   fmt.Sprintf("/%s/stats", abv),
		StatsUiLink: fmt.Sprintf("/%s/stats/ui", abv),
	}
}

func CreateHandlers(d dao.ShortUrlDao, s *status.SimpleStatus, id string, otel *telemetry.Metrics) Handlers {
	return Handlers{dao: d, metrics: metrics{}, otelMetrics: otel, startTime: time.Now(), status: s, id: id}
}

func (h *Handlers) getHandler(c *echo.Context) error {
	atomic.AddUint64(&h.metrics.Redirects, 1)
	h.recordOtelCounter(c.Request().Context(), "redirect")

	abv := c.Param("abv")
	u, err := h.dao.GetUrl(abv)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error getting redirect: %v", err))
	}

	if u == "" {
		return c.String(http.StatusNotFound, "No link found")
	}

	http.Redirect(c.Response(), c.Request(), u, http.StatusFound)
	return nil
}

func (h *Handlers) statsHandler(c *echo.Context) error {
	atomic.AddUint64(&h.metrics.UrlStats, 1)
	h.recordOtelCounter(c.Request().Context(), "stats")

	abv := c.Param("abv")
	stats, err := h.dao.GetStats(abv)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error getting stats: %v", err))
	}

	if stats.Abbreviation == "" {
		return c.String(http.StatusNotFound, "No link found")
	}

	return c.JSON(http.StatusOK, stats)
}

func (h *Handlers) addHandler(c *echo.Context) error {
	atomic.AddUint64(&h.metrics.NewUrls, 1)
	h.recordOtelCounter(c.Request().Context(), "create")

	var u string

	if err := json.NewDecoder(c.Request().Body).Decode(&u); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error parsing url: %v", err))
	}

	if u == "" {
		return c.String(http.StatusBadRequest, "Empty url passed in")
	}

	if parsedUrl, err := url.ParseRequestURI(u); err != nil ||
		parsedUrl.Scheme == "" ||
		parsedUrl.Host == "" {
		return c.String(http.StatusBadRequest, "Invalid url passed in")
	}

	abv, _ := h.dao.GetAbv(u)
	if abv != "" {
		r := createReturn(abv)
		return c.JSON(http.StatusOK, r)
	}

	abv, err := dao.CreateAbbreviation(u, h.dao)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error creating abbreviation: %v", err))
	}

	if err := h.dao.Save(abv, u); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error saving url: %v", err))
	}

	r := createReturn(abv)
	return c.JSON(http.StatusOK, r)
}

func (h *Handlers) deleteHandler(c *echo.Context) error {
	atomic.AddUint64(&h.metrics.Deletes, 1)
	h.recordOtelCounter(c.Request().Context(), "delete")

	abv := c.Param("abv")
	err := h.dao.DeleteAbv(abv)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error deleting: %v", err))
	}

	return c.JSON(http.StatusOK, "deleted")
}

func (h *Handlers) statsUiHandler(c *echo.Context) error {
	abv := c.Param("abv")
	stats, err := h.dao.GetStats(abv)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error getting stats: %v", err))
	}

	if stats.Abbreviation == "" {
		return c.String(http.StatusNotFound, "No link found")
	}

	tmpl := template.Must(template.ParseFiles("stats.html"))
	return tmpl.Execute(c.Response(), stats)
}

func (h *Handlers) SetUp(e *echo.Echo) {
	e.File("/", "index.html")
	e.File("/favicon.ico", "favicon.ico")
	e.GET(statusPath, h.status.BackgroundHandler)
	e.GET(metricsPath, h.metricsHandler)
	e.GET(statsPath, h.statsHandler)
	e.GET(statsUiPath, h.statsUiHandler)
	e.DELETE(appPath, h.deleteHandler)
	e.GET(appPath, h.getHandler)
	e.POST("/", h.addHandler)

	e.Use(h.statusHitsCounter())
	e.Use(h.otelRequestDuration())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		Skipper: func(c *echo.Context) bool {
			return !env.BoolOrDefault("logrequests", true)
		},
		LogMethod: true,
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			fmt.Printf("method=%s, uri=%s, status=%d\n", v.Method, v.URI, v.Status)
			return nil
		},
	}))
	e.Use(h.idHeader())
}

func (h *Handlers) metricsHandler(c *echo.Context) error {
	atomic.AddUint64(&h.metrics.Metrics, 1)
	m := h.metrics
	m.Uptime = time.Since(h.startTime).String()
	return c.JSON(http.StatusOK, m)
}

func (h *Handlers) statusHitsCounter() echo.MiddlewareFunc {
	// using this mechanism since the status handler is in a different package
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if c.Path() == statusPath {
				atomic.AddUint64(&h.metrics.Status, 1)
			}
			return next(c)
		}
	}
}

// idHeader adds the server's id to the "X-SERVER-ID" response header
func (h *Handlers) idHeader() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Add("x-instance-uuid", h.id)
			return next(c)
		}
	}
}

// recordOtelCounter records a counter increment for the given operation type.
func (h *Handlers) recordOtelCounter(ctx context.Context, operation string) {
	if h.otelMetrics == nil {
		return
	}

	switch operation {
	case "redirect":
		h.otelMetrics.Redirects.Add(ctx, 1)
	case "stats":
		h.otelMetrics.StatsRequests.Add(ctx, 1)
	case "create":
		h.otelMetrics.UrlsCreated.Add(ctx, 1)
	case "delete":
		h.otelMetrics.UrlsDeleted.Add(ctx, 1)
	}
}

// otelRequestDuration returns middleware that records request duration as an OTel histogram.
func (h *Handlers) otelRequestDuration() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if h.otelMetrics == nil {
				return next(c)
			}

			start := time.Now()
			err := next(c)
			duration := float64(time.Since(start).Milliseconds())

			status := http.StatusOK
			if resp, respErr := echo.UnwrapResponse(c.Response()); respErr == nil {
				status = resp.Status
			}

			h.otelMetrics.RequestDuration.Record(
				c.Request().Context(),
				duration,
				metric.WithAttributes(
					attribute.String("method", c.Request().Method),
					attribute.String("path", c.Path()),
					attribute.Int("status", status),
				),
			)

			return err
		}
	}
}
