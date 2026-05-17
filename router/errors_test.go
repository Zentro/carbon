package router

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"carbon/internal/server"
	"carbon/remote"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// runWithAbort executes a one-off Gin handler that calls NewError(err).Abort
// and returns the HTTP recorder so the test can assert status + body.
func runWithAbort(err error) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		NewError(err).Abort(c)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	return w
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v body=%s", err, w.Body.String())
	}
	return body.Error
}

func TestAbort_IpMismatch(t *testing.T) {
	w := runWithAbort(ErrIpMismatch)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if msg := decodeError(t, w); msg == "" {
		t.Error("expected error message in body")
	}
}

func TestAbort_GormNotFound(t *testing.T) {
	w := runWithAbort(gorm.ErrRecordNotFound)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestAbort_RemoteRequestError_NotFound(t *testing.T) {
	w := runWithAbort(forgeRemoteError(t, http.StatusNotFound, "missing"))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestAbort_RemoteRequestError_OtherStatusPassesThrough(t *testing.T) {
	w := runWithAbort(forgeRemoteError(t, http.StatusTeapot, "tea"))
	if w.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", w.Code)
	}
	if got := decodeError(t, w); got != "tea" {
		t.Errorf("error message = %q, want %q (upstream message should pass through)", got, "tea")
	}
}

func TestAbort_KnockTimeout(t *testing.T) {
	w := runWithAbort(server.ErrTimeout)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAbort_Generic(t *testing.T) {
	w := runWithAbort(errors.New("random"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// forgeRemoteError builds a *remote.RequestError whose embedded response
// reports the given status. We go through remote.Response.Error() because
// the response field on RequestError is unexported.
func forgeRemoteError(t *testing.T, status int, msg string) error {
	t.Helper()
	body, _ := json.Marshal(struct {
		Errors []remote.RequestError `json:"errors"`
	}{
		Errors: []remote.RequestError{{Code: "x", Status: status, Message: msg}},
	})

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write(body)
	zw.Close()

	resp := &remote.Response{Response: &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(&buf),
	}}
	return resp.Error()
}
