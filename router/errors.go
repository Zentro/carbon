// Copyright (C) 2022 Rafael Galvan

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package router

import (
	"carbon/remote"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	ErrIpMismatch = errors.New("invalid IP address")
)

// RequestError is a custom error type we'll use for formating errors
// for users.
type RequestError struct {
	err error
}

// NewError will return a RequestError.
func NewError(err error) *RequestError {
	return &RequestError{err: err}
}

// Abort aborts the given HTTP request with the specified status code.
func (e *RequestError) Abort(c *gin.Context) {
	if errors.Is(e.err, ErrIpMismatch) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "You are not authorized access this endpoint from a different device.",
		})
		return
	}

	// Look at the RequestError and determine if it an HTTP error from
	// XenForo so we can process and return differently the error for
	// the user.
	if err := remote.AsRequestError(e.err); err != nil {
		switch err.StatusCode() {
		case http.StatusNotFound:
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "The requested resource could not be found.",
			})
			return
		default:
			c.AbortWithStatusJSON(err.StatusCode(), gin.H{
				"error": err.Message,
			})
			return
		}
	}

	if errors.Is(e.err, gorm.ErrRecordNotFound) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"error": "The requested resource could not be found.",
		})
		return
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": "An unexpected error was encountered while processing this request.",
	})
}
