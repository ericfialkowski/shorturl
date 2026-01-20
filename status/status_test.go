package status

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestNewStatus(t *testing.T) {
	s := NewStatus()

	if s.Code != Unknown {
		t.Errorf("NewStatus().Code = %v, want %v", s.Code, Unknown)
	}
	if s.Message != "initializing" {
		t.Errorf("NewStatus().Message = %v, want %v", s.Message, "initializing")
	}
	if s.Timestamp == "" {
		t.Error("NewStatus().Timestamp is empty")
	}
}

func TestSimpleStatus_Ok(t *testing.T) {
	s := NewStatus()
	s.Ok("all good")

	if s.Code != OK {
		t.Errorf("After Ok(), Code = %v, want %v", s.Code, OK)
	}
	if s.Message != "all good" {
		t.Errorf("After Ok(), Message = %v, want %v", s.Message, "all good")
	}
}

func TestSimpleStatus_Warn(t *testing.T) {
	s := NewStatus()
	s.Warn("warning message")

	if s.Code != Warning {
		t.Errorf("After Warn(), Code = %v, want %v", s.Code, Warning)
	}
	if s.Message != "warning message" {
		t.Errorf("After Warn(), Message = %v, want %v", s.Message, "warning message")
	}
}

func TestSimpleStatus_Critical(t *testing.T) {
	s := NewStatus()
	s.Critical("critical error")

	if s.Code != Critical {
		t.Errorf("After Critical(), Code = %v, want %v", s.Code, Critical)
	}
	if s.Message != "critical error" {
		t.Errorf("After Critical(), Message = %v, want %v", s.Message, "critical error")
	}
}

func TestSimpleStatus_Unknown(t *testing.T) {
	s := NewStatus()
	s.Ok("was ok")
	s.Unknown("unknown state")

	if s.Code != Unknown {
		t.Errorf("After Unknown(), Code = %v, want %v", s.Code, Unknown)
	}
	if s.Message != "unknown state" {
		t.Errorf("After Unknown(), Message = %v, want %v", s.Message, "unknown state")
	}
}

func TestSimpleStatus_Current(t *testing.T) {
	s := NewStatus()
	s.Ok("test message")

	current := s.Current()

	if current.Code != s.Code {
		t.Errorf("Current().Code = %v, want %v", current.Code, s.Code)
	}
	if current.Message != s.Message {
		t.Errorf("Current().Message = %v, want %v", current.Message, s.Message)
	}
	// Timestamp should be updated
	if current.Timestamp == "" {
		t.Error("Current().Timestamp is empty")
	}
}

func TestSimpleStatus_StateTransitions(t *testing.T) {
	s := NewStatus()

	// Unknown -> Ok
	s.Ok("ok")
	if s.Code != OK {
		t.Error("Failed transition Unknown -> Ok")
	}

	// Ok -> Warning
	s.Warn("warn")
	if s.Code != Warning {
		t.Error("Failed transition Ok -> Warning")
	}

	// Warning -> Critical
	s.Critical("crit")
	if s.Code != Critical {
		t.Error("Failed transition Warning -> Critical")
	}

	// Critical -> Ok
	s.Ok("recovered")
	if s.Code != OK {
		t.Error("Failed transition Critical -> Ok")
	}
}

func TestCode_Values(t *testing.T) {
	if OK != 0 {
		t.Errorf("OK = %v, want 0", OK)
	}
	if Warning != 1 {
		t.Errorf("Warning = %v, want 1", Warning)
	}
	if Critical != 2 {
		t.Errorf("Critical = %v, want 2", Critical)
	}
	if Unknown != 3 {
		t.Errorf("Unknown = %v, want 3", Unknown)
	}
}

func TestSimpleStatus_Handler(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewStatus()
	s.Ok("healthy")

	err := s.Handler(c)
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Handler() status = %v, want %v", rec.Code, http.StatusOK)
	}

	var result SimpleStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Code != OK {
		t.Errorf("Handler() response Code = %v, want %v", result.Code, OK)
	}
	if result.Message != "healthy" {
		t.Errorf("Handler() response Message = %v, want %v", result.Message, "healthy")
	}
}

func TestSimpleStatus_BackgroundHandler(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewStatus()
	s.Warn("background check failed")

	err := s.BackgroundHandler(c)
	if err != nil {
		t.Fatalf("BackgroundHandler() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("BackgroundHandler() status = %v, want %v", rec.Code, http.StatusOK)
	}

	var result SimpleStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Code != Warning {
		t.Errorf("BackgroundHandler() response Code = %v, want %v", result.Code, Warning)
	}
}

func TestSimpleStatus_JSONSerialization(t *testing.T) {
	s := NewStatus()
	s.Ok("test")

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, ok := result["status_code"]; !ok {
		t.Error("JSON missing 'status_code' field")
	}
	if _, ok := result["status_msg"]; !ok {
		t.Error("JSON missing 'status_msg' field")
	}
	if _, ok := result["timestamp"]; !ok {
		t.Error("JSON missing 'timestamp' field")
	}
}
