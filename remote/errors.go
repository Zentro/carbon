package remote

import (
	"fmt"
	"net/http"

	"emperror.dev/errors"
)

type RequestErrors struct {
	Errors []RequestError `json:"errors"`
}

type RequestError struct {
	response *http.Response
	Code     string `json:"code"`
	Status   int    `json:"status"`
	Message  string `json:"message"`
}

func (re *RequestError) StatusCode() int {
	return re.response.StatusCode
}

func IsRequestError(err error) bool {
	var rerr *RequestError
	if err == nil {
		return false
	}
	return errors.As(err, &rerr)
}

func AsRequestError(err error) *RequestError {
	if err == nil {
		return nil
	}
	var rerr *RequestError
	if errors.As(err, &rerr) {
		return rerr
	}
	return nil
}

func (re *RequestError) Error() string {
	c := 0
	if re.response != nil {
		c = re.response.StatusCode
	}

	return fmt.Sprintf("HTTP error response: %s: %s (HTTP/%d)", re.Code, re.Message, c)
}
