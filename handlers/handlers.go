package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shorturl/dao"

	"github.com/gorilla/mux"
)

const contentType string = "Content-Type"
const appJson string = "application/json"
const AppPath string = "/{app}"
const ListPath string = "/list"

type Handlers struct {
	dao dao.ShorturlDao
}

func CreateHandlers(d dao.ShorturlDao) Handlers {
	return Handlers{dao: d}
}

func (h *Handlers) ListPoisonedHandler(writer http.ResponseWriter, _ *http.Request) {
	apps, err := h.dao.List()
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error getting poisoned apps: %v", err)
		return
	}

	writer.Header().Add(contentType, appJson)
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(apps); err != nil {
		logErr(err)
	}

}

func (h *Handlers) PoisonedHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	app := vars["app"]
	exists, err := h.dao.Exists(app)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error seeing if app is poisoned: %v", err)
		return
	}

	writer.Header().Add(contentType, appJson)
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode(exists); err != nil {
		logErr(err)
	}
}

func (h *Handlers) UnPoisoningHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	app := vars["app"]
	err := h.dao.Delete(app)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error un-poisoning: %v", err)
		return
	}

	writer.Header().Add(contentType, appJson)
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode("deleted"); err != nil {
		logErr(err)
	}
}

func (h *Handlers) PoisoningHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	app := vars["app"]
	err := h.dao.Save(app)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error poisoning: %v", err)
		return
	}

	writer.Header().Add(contentType, appJson)
	writer.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(writer).Encode("added"); err != nil {
		logErr(err)
	}
}

func (h *Handlers) SetUp(r *mux.Router) {
	r.HandleFunc(AppPath, h.PoisoningHandler).Methods("PUT")
	r.HandleFunc(AppPath, h.UnPoisoningHandler).Methods("DELETE")
	r.HandleFunc(ListPath, h.ListPoisonedHandler).Methods("GET")
	r.HandleFunc(AppPath, h.PoisonedHandler).Methods("GET")
}

func logErr(err error) {
	log.Printf("Couldn't encode/write status: %v", err)
}
