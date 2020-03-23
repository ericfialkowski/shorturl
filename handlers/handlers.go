package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"shorturl/dao"
	"shorturl/environment"
	"shorturl/status"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
)

const (
	contentType string = "Content-Type"
	appJson     string = "application/json"
	appPath     string = "/{abv}"
	statsPath   string = "/{abv}/stats"
	statsUiPath string = "/{abv}/stats/ui"
	metricsPath string = "/diag/metrics"
	statusPath  string = "/diag/status"
)

type (
	Handlers struct {
		dao       dao.ShortUrlDao
		metrics   metrics
		startTime time.Time
		status    status.SimpleStatus
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

func CreateHandlers(d dao.ShortUrlDao, s status.SimpleStatus) Handlers {
	return Handlers{dao: d, metrics: metrics{}, startTime: time.Now(), status: s}
}

func (h *Handlers) getHandler(writer http.ResponseWriter, request *http.Request) {
	atomic.AddUint64(&h.metrics.Redirects, 1)
	vars := mux.Vars(request)
	abv := vars["abv"]
	u, err := h.dao.GetUrl(abv)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(writer, "Error getting redirect: %v", err)
		return
	}

	if u == "" {
		writer.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(writer, "No link found")
		return
	}

	http.Redirect(writer, request, u, http.StatusFound)
}

func (h *Handlers) statsHandler(writer http.ResponseWriter, request *http.Request) {
	atomic.AddUint64(&h.metrics.UrlStats, 1)
	vars := mux.Vars(request)
	abv := vars["abv"]
	stats, err := h.dao.GetStats(abv)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(writer, "Error getting stats: %v", err)
		return
	}

	if stats.Abbreviation == "" {
		writer.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(writer, "No link found")
		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Header().Add(contentType, appJson)

	if err := json.NewEncoder(writer).Encode(stats); err != nil {
		logJsonError(err)
	}
}

func (h *Handlers) addHandler(writer http.ResponseWriter, request *http.Request) {
	atomic.AddUint64(&h.metrics.NewUrls, 1)
	var u string

	if err := json.NewDecoder(request.Body).Decode(&u); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(writer, "Error parsing url: %v", err)
		return
	}

	if u == "" {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(writer, "Empty Url Passed In")
		return
	}

	if parsedUrl, err := url.ParseRequestURI(u); err != nil ||
		parsedUrl.Scheme == "" ||
		parsedUrl.Host == "" {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(writer, "Invalid Url Passed In")
		return
	}

	abv, _ := h.dao.GetAbv(u)
	if abv != "" {
		writer.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(writer).Encode(createReturn(abv)); err != nil {
			logJsonError(err)
		}
		return
	}

	abv, err := dao.CreateAbbreviation(u, h.dao)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(writer, "Error creating abbreviation: %v", err)
		return
	}

	if err := h.dao.Save(abv, u); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(writer, "Error saving url: %v", err)
		return
	}

	writer.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(writer).Encode(createReturn(abv)); err != nil {
		logJsonError(err)
	}
}

func (h *Handlers) deleteHandler(writer http.ResponseWriter, request *http.Request) {
	atomic.AddUint64(&h.metrics.Deletes, 1)
	vars := mux.Vars(request)
	abv := vars["abv"]
	err := h.dao.DeleteAbv(abv)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(writer, "Error deleting: %v", err)
		return
	}

	writer.Header().Add(contentType, appJson)
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode("deleted"); err != nil {
		logJsonError(err)
	}
}

func (h *Handlers) landingPageHandler(writer http.ResponseWriter, _ *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	if err := tmpl.Execute(writer, nil); err != nil {
		logErr(err)
	}
}

func (h *Handlers) statsUiHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	abv := vars["abv"]
	stats, err := h.dao.GetStats(abv)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(writer, "Error getting stats: %v", err)
		return
	}

	if stats.Abbreviation == "" {
		writer.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(writer, "No link found")
		return
	}

	tmpl := template.Must(template.ParseFiles("stats.html"))
	if err := tmpl.Execute(writer, stats); err != nil {
		logErr(err)
	}
}

func (h *Handlers) SetUp(router *mux.Router) {
	router.HandleFunc("/", h.landingPageHandler).Methods(http.MethodGet)
	router.HandleFunc(statusPath, h.status.BackgroundHandler)
	router.HandleFunc(metricsPath, h.metricsHandler).Methods(http.MethodGet)
	router.HandleFunc(statsPath, h.statsHandler).Methods(http.MethodGet)
	router.HandleFunc(statsUiPath, h.statsUiHandler).Methods(http.MethodGet)
	router.HandleFunc(appPath, h.deleteHandler).Methods(http.MethodDelete)
	router.HandleFunc(appPath, h.getHandler).Methods(http.MethodGet)
	router.HandleFunc("/", h.addHandler).Methods(http.MethodPost)
	router.Use(logWrapper)
	router.Use(h.hitsCounterWrapper)
}

func (h *Handlers) metricsHandler(writer http.ResponseWriter, _ *http.Request) {
	atomic.AddUint64(&h.metrics.Metrics, 1)
	writer.Header().Add(contentType, appJson)
	writer.WriteHeader(http.StatusOK)
	m := h.metrics
	m.Uptime = time.Since(h.startTime).String()
	if err := json.NewEncoder(writer).Encode(m); err != nil {
		logJsonError(err)
	}
}

func logErr(err error) {
	log.Printf("Couldn't send output: %v", err)
}

func logJsonError(err error) {
	log.Printf("Couldn't encode/write json: %v", err)
}

func logWrapper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if environment.GetEnvBoolOrDefault("logrequests", false) {
			log.Printf("access:  %s - %s\n", request.Method, request.RequestURI)
		}
		next.ServeHTTP(writer, request)
	})
}

func (h *Handlers) hitsCounterWrapper(next http.Handler) http.Handler {
	// using this mechanism since the status handler is in a different package
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.RequestURI == statusPath {
			atomic.AddUint64(&h.metrics.Status, 1)
		}
		next.ServeHTTP(writer, request)
	})
}
