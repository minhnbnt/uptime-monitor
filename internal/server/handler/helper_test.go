package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

func newGinContext(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	return newGinContextWithUser(method, path, body, 0)
}

func newGinContextWithUser(method, path, body string, userID uint) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	if userID != 0 {
		c.Set(UserIDKey, userID)
	}

	return c, w
}

func parseJSON(w *httptest.ResponseRecorder, target any) {
	data, _ := io.ReadAll(w.Body)
	json.Unmarshal(data, target)
}
