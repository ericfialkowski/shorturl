package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ericfialkowski/shorturl/dao"
	"github.com/ericfialkowski/shorturl/status"
	"github.com/labstack/echo/v5"
)

func setupTestHandlers() (*Handlers, *echo.Echo) {
	db := dao.CreateMemoryDB()
	s := status.NewStatus()
	s.Ok("test")
	h := CreateHandlers(db, s, "test-id", nil)
	e := echo.New()
	h.SetUp(e)
	return &h, e
}

func TestCreateHandlers(t *testing.T) {
	db := dao.CreateMemoryDB()
	s := status.NewStatus()
	h := CreateHandlers(db, s, "test-id", nil)

	if h.id != "test-id" {
		t.Errorf("CreateHandlers().id = %v, want %v", h.id, "test-id")
	}
}

func TestHandlers_AddHandler_ValidURL(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`"https://example.com"`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.addHandler(c)
	if err != nil {
		t.Fatalf("addHandler() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("addHandler() status = %v, want %v", rec.Code, http.StatusOK)
	}

	var result urlReturn
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Abv == "" {
		t.Error("addHandler() returned empty abbreviation")
	}
	if result.UrlLink == "" {
		t.Error("addHandler() returned empty UrlLink")
	}
}

func TestHandlers_AddHandler_EmptyURL(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`""`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.addHandler(c)
	if err != nil {
		t.Fatalf("addHandler() error = %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("addHandler() with empty URL status = %v, want %v", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlers_AddHandler_InvalidURL(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	testCases := []struct {
		name string
		url  string
	}{
		{"no scheme", `"example.com"`},
		{"no host", `"https://"`},
		{"invalid format", `"not-a-url"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.url))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.addHandler(c)
			if err != nil {
				t.Fatalf("addHandler() error = %v", err)
			}

			if rec.Code != http.StatusBadRequest {
				t.Errorf("addHandler() with %s status = %v, want %v", tc.name, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandlers_AddHandler_DuplicateURL(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	url := `"https://duplicate.com"`

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	_ = h.addHandler(c1)

	var result1 urlReturn
	_ = json.Unmarshal(rec1.Body.Bytes(), &result1)

	// Second request with same URL
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	_ = h.addHandler(c2)

	var result2 urlReturn
	_ = json.Unmarshal(rec2.Body.Bytes(), &result2)

	// Should return same abbreviation
	if result1.Abv != result2.Abv {
		t.Errorf("Duplicate URL returned different abbreviations: %v vs %v", result1.Abv, result2.Abv)
	}
}

func TestHandlers_GetHandler_Found(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	// First, add a URL
	_ = h.dao.Save("test1", "https://redirect.com")

	req := httptest.NewRequest(http.MethodGet, "/test1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:abv")
	c.SetPathValues(echo.PathValues{{Name: "abv", Value: "test1"}})

	err := h.getHandler(c)
	if err != nil {
		t.Fatalf("getHandler() error = %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Errorf("getHandler() status = %v, want %v", rec.Code, http.StatusFound)
	}

	location := rec.Header().Get("Location")
	if location != "https://redirect.com" {
		t.Errorf("getHandler() Location = %v, want %v", location, "https://redirect.com")
	}
}

func TestHandlers_GetHandler_NotFound(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:abv")
	c.SetPathValues(echo.PathValues{{Name: "abv", Value: "nonexistent"}})

	err := h.getHandler(c)
	if err != nil {
		t.Fatalf("getHandler() error = %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("getHandler() for missing URL status = %v, want %v", rec.Code, http.StatusNotFound)
	}
}

func TestHandlers_StatsHandler_Found(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	_ = h.dao.Save("stat1", "https://stats.com")

	req := httptest.NewRequest(http.MethodGet, "/stat1/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:abv/stats")
	c.SetPathValues(echo.PathValues{{Name: "abv", Value: "stat1"}})

	err := h.statsHandler(c)
	if err != nil {
		t.Fatalf("statsHandler() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("statsHandler() status = %v, want %v", rec.Code, http.StatusOK)
	}

	var result dao.ShortUrl
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Abbreviation != "stat1" {
		t.Errorf("statsHandler() Abbreviation = %v, want %v", result.Abbreviation, "stat1")
	}
}

func TestHandlers_StatsHandler_NotFound(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	req := httptest.NewRequest(http.MethodGet, "/missing/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:abv/stats")
	c.SetPathValues(echo.PathValues{{Name: "abv", Value: "missing"}})

	err := h.statsHandler(c)
	if err != nil {
		t.Fatalf("statsHandler() error = %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("statsHandler() for missing URL status = %v, want %v", rec.Code, http.StatusNotFound)
	}
}

func TestHandlers_DeleteHandler(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	_ = h.dao.Save("del1", "https://delete.com")

	req := httptest.NewRequest(http.MethodDelete, "/del1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:abv")
	c.SetPathValues(echo.PathValues{{Name: "abv", Value: "del1"}})

	err := h.deleteHandler(c)
	if err != nil {
		t.Fatalf("deleteHandler() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("deleteHandler() status = %v, want %v", rec.Code, http.StatusOK)
	}

	// Verify deletion
	url, _ := h.dao.GetUrl("del1")
	if url != "" {
		t.Error("deleteHandler() did not delete the URL")
	}
}

func TestHandlers_MetricsHandler(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	req := httptest.NewRequest(http.MethodGet, "/diag/metrics", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.metricsHandler(c)
	if err != nil {
		t.Fatalf("metricsHandler() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("metricsHandler() status = %v, want %v", rec.Code, http.StatusOK)
	}

	var result metrics
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Uptime == "" {
		t.Error("metricsHandler() Uptime is empty")
	}
}

func TestHandlers_MetricsIncrement(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	_ = h.dao.Save("inc1", "https://increment.com")

	// Make a redirect request
	req := httptest.NewRequest(http.MethodGet, "/inc1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:abv")
	c.SetPathValues(echo.PathValues{{Name: "abv", Value: "inc1"}})
	_ = h.getHandler(c)

	if h.metrics.Redirects != 1 {
		t.Errorf("After redirect, metrics.Redirects = %v, want 1", h.metrics.Redirects)
	}
}

func TestHandlers_IdHeader(t *testing.T) {
	h, e := setupTestHandlers()
	defer h.dao.Cleanup()

	middleware := h.idHeader()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	if err != nil {
		t.Fatalf("idHeader middleware error = %v", err)
	}

	header := rec.Header().Get("x-instance-uuid")
	if header != "test-id" {
		t.Errorf("idHeader() set header = %v, want %v", header, "test-id")
	}
}

func TestCreateReturn(t *testing.T) {
	result := createReturn("abc")

	if result.Abv != "abc" {
		t.Errorf("createReturn().Abv = %v, want %v", result.Abv, "abc")
	}
	if result.UrlLink != "/abc" {
		t.Errorf("createReturn().UrlLink = %v, want %v", result.UrlLink, "/abc")
	}
	if result.StatsLink != "/abc/stats" {
		t.Errorf("createReturn().StatsLink = %v, want %v", result.StatsLink, "/abc/stats")
	}
	if result.StatsUiLink != "/abc/stats/ui" {
		t.Errorf("createReturn().StatsUiLink = %v, want %v", result.StatsUiLink, "/abc/stats/ui")
	}
}
