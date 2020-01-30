package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"shorturl/dao"

	"github.com/gorilla/mux"
)

const contentType string = "Content-Type"
const appJson string = "application/json"
const AppPath string = "/{abv}"

type Handlers struct {
	dao dao.ShortUrlDao
}

func CreateHandlers(d dao.ShortUrlDao) Handlers {
	return Handlers{dao: d}
}

func (h *Handlers) GetHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	abv := vars["abv"]
	url, err := h.dao.GetUrl(abv)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error getting redirect: %v", err)
		return
	}

	if len(url) == 0 {
		writer.WriteHeader(http.StatusNotFound)
		fmt.Fprint(writer, "No link found")
		return
	}

	http.Redirect(writer, request, url, http.StatusFound)
}

func (h *Handlers) AddHandler(writer http.ResponseWriter, request *http.Request) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error parsing url: %v", err)
		return
	}

	url := string(body)
	if len(url) == 0 {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, "Empty Url Passed In")
		return
	}

	abv, err := h.dao.GetAbv(url)
	if len(abv) > 0 {
		writer.WriteHeader(http.StatusOK)
		fmt.Fprintf(writer, "%s%s%s", request.Host, request.RequestURI, abv)
		return
	}

	abv, err = dao.CreateAbbreviation(url, h.dao)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error creating abbreviation: %v", err)
		return
	}

	err = h.dao.Save(abv, url)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error saving url: %v", err)
		return
	}

	writer.WriteHeader(http.StatusCreated)
	fmt.Fprintf(writer, "%s%s%s", request.Host, request.RequestURI, abv)
}

func (h *Handlers) DeleteHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	abv := vars["abv"]
	err := h.dao.DeleteAbv(abv)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error deleting: %v", err)
		return
	}

	writer.Header().Add(contentType, appJson)
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode("deleted"); err != nil {
		logErr(err)
	}
}

func (h *Handlers) SetUp(r *mux.Router) {
	r.HandleFunc(AppPath, h.DeleteHandler).Methods("DELETE")
	r.HandleFunc(AppPath, h.GetHandler).Methods("GET")
	r.HandleFunc("/", h.AddHandler).Methods("POST")
}

func logErr(err error) {
	log.Printf("Couldn't encode/write status: %v", err)
}
