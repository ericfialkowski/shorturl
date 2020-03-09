package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"shorturl/dao"
	"shorturl/environment"

	"github.com/gorilla/mux"
)

const contentType string = "Content-Type"
const appJson string = "application/json"
const AppPath string = "/{abv}"
const StatsPath string = "/{abv}/stats"

type Handlers struct {
	dao dao.ShortUrlDao
}

func CreateHandlers(d dao.ShortUrlDao) Handlers {
	return Handlers{dao: d}
}

func (h *Handlers) getHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	abv := vars["abv"]
	u, err := h.dao.GetUrl(abv)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(writer, "Error getting redirect: %v", err)
		return
	}

	if len(u) == 0 {
		writer.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(writer, "No link found")
		return
	}

	http.Redirect(writer, request, u, http.StatusFound)
}

func (h *Handlers) statsHandler(writer http.ResponseWriter, request *http.Request) {
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
	var u string

	if err := json.NewDecoder(request.Body).Decode(&u); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(writer, "Error parsing url: %v", err)
		return
	}

	if len(u) == 0 {
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
	if len(abv) > 0 {
		writer.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(writer, "%s%s%s", request.Host, request.RequestURI, abv)
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
	_, _ = fmt.Fprintf(writer, "%s%s%s", request.Host, request.RequestURI, abv)
}

func (h *Handlers) deleteHandler(writer http.ResponseWriter, request *http.Request) {
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

func (h *Handlers) SetUp(router *mux.Router) {
	router.HandleFunc(StatsPath, h.statsHandler).Methods(http.MethodGet)
	router.HandleFunc(AppPath, h.deleteHandler).Methods(http.MethodDelete)
	router.HandleFunc(AppPath, h.getHandler).Methods(http.MethodGet)
	router.HandleFunc("/", h.addHandler).Methods(http.MethodPost)
	router.Use(logWrapper)
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
