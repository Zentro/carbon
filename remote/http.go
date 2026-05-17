// Copyright (C) 2022-2023 Rafael Galvan <rafael.galvan@rigsofrods.org>

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

package remote

import (
	"bytes"
	"carbon/domain"
	"carbon/system"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"emperror.dev/errors"
	"github.com/cenkalti/backoff/v4"
	jsoniter "github.com/json-iterator/go"
)

type Client interface {
	GetResources(ctx context.Context) ([]domain.Resource, error)
	GetResource(ctx context.Context, rid string) (domain.Resource, error)
	GetResourceCategories(ctx context.Context) ([]domain.ResourceCategory, TreeMap, error)
	GetResourceCategory(ctx context.Context) (domain.ResourceCategory, error)
	GetResourceReviews(ctx context.Context, rid string) ([]domain.ResourceReview, error)
	GetResourceVersions(ctx context.Context, rid string) ([]domain.ResourceVersion, error)
	GetResourceVersion(ctx context.Context, vid string) (domain.ResourceVersion, error)
	GetUser(ctx context.Context, uid int) (domain.User, error)
	ValidateUserAuthCredentials(ctx context.Context, data interface{}) (RawUserAuthResponse, error)
}

// authMode selects which credential a single XF request will present.
type authMode int

const (
	// authData presents the non-super DataKey. Used for all reads
	// (resources, users, categories, reviews) — anything that doesn't
	// need super-user scopes.
	authData authMode = iota
	// authBridge presents the super-user BridgeKey. Used ONLY by the
	// /bridge/auth password-validation call, which requires the
	// super-user-only `bridge` and `auth` scopes.
	authBridge
)

type client struct {
	httpClient  *http.Client
	baseUrl     string
	bridgeKey   string
	dataKey     string
	maxAttempts int
}

// NewClient returns an HTTP request client for the XenForo REST API. It
// presents the narrow data key by default and only falls back to the
// super-user bridge key on the explicit bridge-auth path.
func NewClient(base, bridgeKey, dataKey string) Client {
	c := client{
		baseUrl:     base,
		httpClient:  &http.Client{Timeout: time.Second * 15},
		bridgeKey:   bridgeKey,
		dataKey:     dataKey,
		maxAttempts: 0,
	}
	return &c
}

// Get will make an HTTP GET request authenticated with the data key.
func (c *client) Get(ctx context.Context, path string, query q, headers q) (*Response, error) {
	return c.requestWithRetries(ctx, http.MethodGet, path, authData, nil, headers, func(r *http.Request) {
		q := r.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		r.URL.RawQuery = q.Encode()
	})
}

// Post will make an HTTP POST request authenticated with the data key.
func (c *client) Post(ctx context.Context, path string, body url.Values, headers q) (*Response, error) {
	return c.requestWithRetries(ctx, http.MethodPost, path, authData, bytes.NewBufferString(body.Encode()), headers)
}

// PostBridge makes an HTTP POST authenticated with the XF super-user key.
// Reserved for /bridge/auth — never call this for anything else.
func (c *client) PostBridge(ctx context.Context, path string, body url.Values, headers q) (*Response, error) {
	return c.requestWithRetries(ctx, http.MethodPost, path, authBridge, bytes.NewBufferString(body.Encode()), headers)
}

// applyAuth attaches the credential appropriate to the chosen auth mode.
func (c *client) applyAuth(_ context.Context, req *http.Request, mode authMode) error {
	switch mode {
	case authBridge:
		req.Header.Set("XF-Api-Key", c.bridgeKey)
		return nil
	case authData:
		req.Header.Set("XF-Api-Key", c.dataKey)
		return nil
	default:
		return errors.Errorf("remote/http: unknown auth mode %d", mode)
	}
}

// request will make an HTTP request and execute it once with our required headers.
func (c *client) request(ctx context.Context, method string, path string, mode authMode, body io.Reader, headers q, opts ...func(r *http.Request)) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseUrl+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", fmt.Sprintf("carbon/v%s", system.Version))
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // https://xenforo.com/docs/dev/rest-api/#accessing-the-api
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	if err := c.applyAuth(ctx, req, mode); err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	for _, o := range opts {
		o(req)
	}

	logHttpRequests(req)

	res, err := c.httpClient.Do(req)
	return &Response{res}, err
}

// requestWithRetries will make an HTTP request against the API using an exponential
// backoff if an error is returned. Any type of 400 error will not be retried.
func (c *client) requestWithRetries(ctx context.Context, method string, path string, mode authMode, body io.Reader, headers q, opts ...func(r *http.Request)) (*Response, error) {
	var res *Response
	var lastError error
	err := backoff.Retry(func() error {
		r, err := c.request(ctx, method, path, mode, body, headers, opts...)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return backoff.Permanent(err)
			}

			lastError = err
			return err
		}
		res = r
		if r.HasError() {
			defer r.Body.Close()
			// Don't keep trying to spam the endpoint if the error is not likely
			// to change.
			if r.StatusCode == http.StatusForbidden ||
				r.StatusCode == http.StatusTooManyRequests ||
				r.StatusCode == http.StatusUnauthorized {
				return backoff.Permanent(r.Error())
			}
			return backoff.Permanent(r.Error())
		}
		return nil
	}, c.backoff(ctx))
	if err != nil {
		if v, ok := err.(*backoff.PermanentError); ok {
			return nil, v.Unwrap()
		}
		if lastError != nil {
			return nil, lastError
		}
		return nil, err
	}
	return res, nil
}

// backoff returns an exponential backoff function to be used with remote API
// requests. This will allow an API to call to be executed approximately 5 times
// before it is finally reported back as an error.
//
// The MaxElapsedTime is arbitrarily set at 10 as this seems like a sweet spot
// for how long we should reasonably wait until we report back an error.
func (c *client) backoff(ctx context.Context) backoff.BackOffContext {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Second * 10
	if c.maxAttempts > 0 {
		return backoff.WithContext(backoff.WithMaxRetries(b, uint64(c.maxAttempts)), ctx)
	}
	return backoff.WithContext(b, ctx)
}

type Response struct {
	*http.Response
}

// HasError will check between the HTTP status ranges of 300, 400, and 500.
func (r *Response) HasError() bool {
	if r.Response == nil {
		return false
	}

	return r.StatusCode >= 300 || r.StatusCode < 200
}

// Error returns the error message from the API. If there is no error we will
// resort a HTTP 503 Service Unavaialble indicating the API is unavaialble.
func (r *Response) Error() error {
	if !r.HasError() {
		return nil
	}

	var errs RequestErrors
	_ = r.BindJSON(&errs)

	e := &RequestError{
		Code:    "_MissingResponseCode",
		Status:  r.StatusCode,
		Message: "No error response returned from the API endpoint.",
	}
	if len(errs.Errors) > 0 {
		e = &errs.Errors[0]
	}

	e.response = r.Response

	return errors.WithStackDepth(e, 1)
}

// Read will read the response body of the HTTP request. We will try to read
// as if the the body was compressed.
func (r *Response) Read() ([]byte, error) {
	var b []byte
	if r.Response == nil {
		return nil, errors.New("remote: attempting to read missing response")
	}
	if r.Response.Body != nil {
		// Read the compressed body and pass it off to be read.
		d, _ := gzip.NewReader(r.Response.Body)
		defer d.Close()
		b, _ = io.ReadAll(d)
	}
	r.Response.Body = io.NopCloser(bytes.NewBuffer(b))
	return b, nil
}

func (r *Response) BindJSON(v interface{}) error {
	b, err := r.Read()
	if err != nil {
		return err
	}
	if err := jsoniter.Unmarshal(b, &v); err != nil {
		return errors.Wrap(err, "remote: could not unmarshal response")
	}
	return nil
}

// Logs the request if and only if we're running in debug mode. Redacts any
// header that carries a credential so we never leak super keys or bearer
// tokens through debug logs.
func logHttpRequests(req *http.Request) {
	headers := make(map[string][]string)
	for k, v := range req.Header {
		if isSensitiveHeader(k) && len(v) > 0 && len(v[0]) > 0 {
			headers[k] = []string{"(redacted)"}
			continue
		}
		headers[k] = v
	}

	slog.Debug("request to external HTTP endpoint",
		"method", req.Method,
		"endpoint", req.URL.String(),
		"headers", headers,
	)
}

func isSensitiveHeader(name string) bool {
	switch name {
	case "Xf-Api-Key", "Authorization", "Proxy-Authorization", "Cookie":
		return true
	}
	return false
}
