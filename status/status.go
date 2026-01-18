package status

import (
	"github.com/labstack/echo/v5"
	"net/http"
	"time"
)

type (
	Code int

	SimpleStatus struct {
		Code      Code   `json:"status_code"`
		Message   string `json:"status_msg"`
		Timestamp string `json:"timestamp"`
	}
)

const (
	OK Code = iota
	Warning
	Critical
	Unknown
)

func NewStatus() *SimpleStatus {
	return &SimpleStatus{Code: Unknown, Message: "initializing", Timestamp: currentTimestamp()}
}

func (s *SimpleStatus) newStatus(code Code, message string) {
	s.Code = code
	s.Message = message
	s.Timestamp = currentTimestamp()
}

func (s *SimpleStatus) Current() SimpleStatus {
	s.Timestamp = currentTimestamp()
	return *s
}

func currentTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05") // Note to self - wtf format?
}

func (s *SimpleStatus) Ok(message string) {
	s.newStatus(OK, message)
}

func (s *SimpleStatus) Warn(message string) {
	s.newStatus(Warning, message)
}

func (s *SimpleStatus) Critical(message string) {
	s.newStatus(Critical, message)
}

func (s *SimpleStatus) Unknown(message string) {
	s.newStatus(Unknown, message)
}

/*
Handler is used for a slowly changing status where we want to automatically update the timestamp to the current request time
*/
func (s *SimpleStatus) Handler(c *echo.Context) error {
	return c.JSON(http.StatusOK, s.Current())
}

/*
BackgroundHandler is used when there will be a background process that updates the status,
and we want to see the timestamp of when the background task ran last
*/
func (s *SimpleStatus) BackgroundHandler(c *echo.Context) error {
	return c.JSON(http.StatusOK, s)
}
