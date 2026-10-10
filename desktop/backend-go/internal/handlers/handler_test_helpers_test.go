package handlers

// handler_test_helpers_test.go — shared test utility functions used across
// multiple test files in the handlers package.

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// runHandlerCatchPanic runs a gin handler with the given body and recovers from
// any panic that occurs (e.g. from nil pool/queries when testing binding only).
// Returns the HTTP status code written by the handler, or 500 if a panic occurred.
func runHandlerCatchPanic(t *testing.T, body []byte, handler func(c *gin.Context)) (status int) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	defer func() {
		if rec := recover(); rec != nil {
			// Panic from nil pool — count as server error, not a binding error.
			status = http.StatusInternalServerError
		}
	}()

	r.POST("/test", handler)

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(http.MethodPost, "/test", nil)
	}

	r.ServeHTTP(w, req)
	return w.Code
}
