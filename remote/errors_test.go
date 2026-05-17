package remote

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestRequestError_StatusCode(t *testing.T) {
	re := &RequestError{
		response: &http.Response{StatusCode: 418},
		Code:     "teapot",
		Message:  "I'm a teapot",
	}
	if re.StatusCode() != 418 {
		t.Errorf("StatusCode = %d, want 418", re.StatusCode())
	}
}

func TestRequestError_ErrorString(t *testing.T) {
	re := &RequestError{
		response: &http.Response{StatusCode: 404},
		Code:     "not_found",
		Message:  "Missing",
	}
	got := re.Error()
	for _, want := range []string{"not_found", "Missing", "404"} {
		if !strings.Contains(got, want) {
			t.Errorf("Error() = %q, missing %q", got, want)
		}
	}
}

func TestRequestError_Error_NilResponse(t *testing.T) {
	re := &RequestError{Code: "c", Message: "m"}
	got := re.Error()
	// nil response should fall back to status 0, not panic.
	if !strings.Contains(got, "HTTP/0") {
		t.Errorf("Error() with nil response should include HTTP/0, got %q", got)
	}
}

func TestIsRequestError(t *testing.T) {
	if IsRequestError(nil) {
		t.Error("IsRequestError(nil) should be false")
	}
	if IsRequestError(errors.New("generic")) {
		t.Error("IsRequestError(plain) should be false")
	}
	re := &RequestError{Code: "x"}
	if !IsRequestError(re) {
		t.Error("IsRequestError(*RequestError) should be true")
	}
}

func TestAsRequestError(t *testing.T) {
	if AsRequestError(nil) != nil {
		t.Error("AsRequestError(nil) should be nil")
	}
	if AsRequestError(errors.New("plain")) != nil {
		t.Error("AsRequestError(plain) should be nil")
	}
	re := &RequestError{Code: "x"}
	got := AsRequestError(re)
	if got != re {
		t.Errorf("AsRequestError should return the underlying *RequestError, got %v", got)
	}
}
