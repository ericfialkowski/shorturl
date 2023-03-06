package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"html/template"
	"net/http"
	"net/url"
	"shorturl/dao"
	"shorturl/environment"
	"shorturl/status"
	"sync/atomic"
	"time"
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
		dao       dao.ShortUrlDao
		metrics   metrics
		startTime time.Time
		status    *status.SimpleStatus
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

func CreateHandlers(d dao.ShortUrlDao, s *status.SimpleStatus) Handlers {
	return Handlers{dao: d, metrics: metrics{}, startTime: time.Now(), status: s}
}

func (h *Handlers) getHandler(c echo.Context) error {
	atomic.AddUint64(&h.metrics.Redirects, 1)
	abv := c.Param("abv")
	u, err := h.dao.GetUrl(abv)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error getting redirect: %v", err))
	}

	if u == "" {
		return c.String(http.StatusNotFound, "No link found")
	}

	http.Redirect(c.Response().Writer, c.Request(), u, http.StatusFound)
	return nil
}

func (h *Handlers) statsHandler(c echo.Context) error {
	atomic.AddUint64(&h.metrics.UrlStats, 1)
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

func (h *Handlers) addHandler(c echo.Context) error {
	atomic.AddUint64(&h.metrics.NewUrls, 1)
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

func (h *Handlers) deleteHandler(c echo.Context) error {
	atomic.AddUint64(&h.metrics.Deletes, 1)
	abv := c.Param("abv")
	err := h.dao.DeleteAbv(abv)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error deleting: %v", err))
	}

	return c.JSON(http.StatusOK, "deleted")
}

func (h *Handlers) statsUiHandler(c echo.Context) error {
	abv := c.Param("abv")
	stats, err := h.dao.GetStats(abv)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error getting stats: %v", err))
	}

	if stats.Abbreviation == "" {
		return c.String(http.StatusNotFound, "No link found")
	}

	tmpl := template.Must(template.ParseFiles("stats.html"))
	return tmpl.Execute(c.Response().Writer, stats)
}

func (h *Handlers) SetUp(e *echo.Echo) {
	e.File("/", "index.html")
	e.GET(statusPath, h.status.BackgroundHandler)
	e.GET(metricsPath, h.metricsHandler)
	e.GET(statsPath, h.statsHandler)
	e.GET(statsUiPath, h.statsUiHandler)
	e.DELETE(appPath, h.deleteHandler)
	e.GET(appPath, h.getHandler)
	e.POST("/", h.addHandler)

	e.Use(h.statusHitsCounter())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return !environment.GetEnvBoolOrDefault("logrequests", true)
		},
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))
}

func (h *Handlers) metricsHandler(c echo.Context) error {
	atomic.AddUint64(&h.metrics.Metrics, 1)
	m := h.metrics
	m.Uptime = time.Since(h.startTime).String()
	return c.JSON(http.StatusOK, m)
}

func (h *Handlers) statusHitsCounter() echo.MiddlewareFunc {
	// using this mechanism since the status handler is in a different package
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Path() == statusPath {
				atomic.AddUint64(&h.metrics.Status, 1)
			}
			return next(c)
		}
	}
}
