package remote

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsSensitiveHeader(t *testing.T) {
	cases := map[string]bool{
		"Xf-Api-Key":          true,
		"Authorization":       true,
		"Proxy-Authorization": true,
		"Cookie":              true,
		"Content-Type":        false,
		"User-Agent":          false,
		"":                    false,
	}
	for name, want := range cases {
		if got := isSensitiveHeader(name); got != want {
			t.Errorf("isSensitiveHeader(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestResponse_HasError(t *testing.T) {
	cases := []struct {
		name string
		resp *Response
		want bool
	}{
		{"nil response", &Response{Response: nil}, false},
		{"200 OK", &Response{Response: &http.Response{StatusCode: 200}}, false},
		{"299", &Response{Response: &http.Response{StatusCode: 299}}, false},
		{"300", &Response{Response: &http.Response{StatusCode: 300}}, true},
		{"404", &Response{Response: &http.Response{StatusCode: 404}}, true},
		{"500", &Response{Response: &http.Response{StatusCode: 500}}, true},
		{"199", &Response{Response: &http.Response{StatusCode: 199}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.resp.HasError(); got != tc.want {
				t.Errorf("HasError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func gzipBytes(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestResponse_Read_DecodesGzip(t *testing.T) {
	body := gzipBytes(t, []byte(`{"hello":"world"}`))
	r := &Response{Response: &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}}

	b, err := r.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(b) != `{"hello":"world"}` {
		t.Errorf("Read returned %q", b)
	}
}

func TestResponse_Read_NilResponse(t *testing.T) {
	r := &Response{Response: nil}
	if _, err := r.Read(); err == nil {
		t.Error("expected error reading nil response")
	}
}

func TestResponse_BindJSON(t *testing.T) {
	body := gzipBytes(t, []byte(`{"name":"alice","age":30}`))
	r := &Response{Response: &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}}

	var got struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	if err := r.BindJSON(&got); err != nil {
		t.Fatalf("BindJSON: %v", err)
	}
	if got.Name != "alice" || got.Age != 30 {
		t.Errorf("BindJSON result mismatch: %+v", got)
	}
}

func TestResponse_BindJSON_InvalidJSON(t *testing.T) {
	body := gzipBytes(t, []byte(`not json`))
	r := &Response{Response: &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}}
	var dst map[string]string
	if err := r.BindJSON(&dst); err == nil {
		t.Error("expected error binding invalid JSON")
	}
}

func TestResponse_Error_PopulatesFromBody(t *testing.T) {
	errBody := gzipBytes(t, []byte(`{"errors":[{"code":"bad_thing","status":400,"message":"boom"}]}`))
	r := &Response{Response: &http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(bytes.NewReader(errBody)),
	}}

	err := r.Error()
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	rerr := AsRequestError(err)
	if rerr == nil {
		t.Fatalf("expected *RequestError, got %T", err)
	}
	if rerr.Code != "bad_thing" || rerr.Message != "boom" {
		t.Errorf("error fields not populated: %+v", rerr)
	}
}

func TestResponse_Error_NoErrorOnSuccess(t *testing.T) {
	r := &Response{Response: &http.Response{StatusCode: 200}}
	if err := r.Error(); err != nil {
		t.Errorf("Error() on 200 should be nil, got %v", err)
	}
}

func TestClient_GetSendsDataKey(t *testing.T) {
	var gotApiKey, gotMethod, gotUserAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotApiKey = r.Header.Get("XF-Api-Key")
		gotMethod = r.Method
		gotUserAgent = r.Header.Get("User-Agent")

		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		w.Write(gzipBytes(t, []byte(`{"ok":true}`)))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "BRIDGE", "DATA").(*client)
	resp, err := c.Get(context.Background(), "/anything", nil, nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if gotApiKey != "DATA" {
		t.Errorf("XF-Api-Key = %q, want DATA (data key, not bridge)", gotApiKey)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.HasPrefix(gotUserAgent, "carbon/") {
		t.Errorf("User-Agent = %q, want carbon/ prefix", gotUserAgent)
	}
}

func TestClient_PostBridgeSendsBridgeKey(t *testing.T) {
	var gotApiKey, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotApiKey = r.Header.Get("XF-Api-Key")
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write(gzipBytes(t, []byte(`{}`)))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "BRIDGE", "DATA").(*client)
	resp, err := c.PostBridge(context.Background(), "/bridge/auth", nil, nil)
	if err != nil {
		t.Fatalf("PostBridge: %v", err)
	}
	defer resp.Body.Close()

	if gotApiKey != "BRIDGE" {
		t.Errorf("XF-Api-Key = %q, want BRIDGE (super-user key for /bridge/auth)", gotApiKey)
	}
	if !strings.Contains(gotContentType, "x-www-form-urlencoded") {
		t.Errorf("Content-Type = %q, want form-urlencoded", gotContentType)
	}
}

func TestClient_GetPropagatesQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Encode()
		w.WriteHeader(http.StatusOK)
		w.Write(gzipBytes(t, []byte(`{}`)))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "B", "D").(*client)
	_, err := c.Get(context.Background(), "/x", q{"page": "3"}, nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotQuery != "page=3" {
		t.Errorf("query = %q, want page=3", gotQuery)
	}
}

func TestClient_Get_ServerErrorBecomesRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write(gzipBytes(t, []byte(`{"errors":[{"code":"not_found","status":404,"message":"nope"}]}`)))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "B", "D").(*client)
	c.maxAttempts = 1 // keep retries tight

	_, err := c.Get(context.Background(), "/x", nil, nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	rerr := AsRequestError(err)
	if rerr == nil {
		t.Fatalf("expected *RequestError, got %T (%v)", err, err)
	}
	if rerr.Code != "not_found" || rerr.Message != "nope" {
		t.Errorf("error fields not populated: %+v", rerr)
	}
}

func TestApplyAuth_UnknownMode(t *testing.T) {
	c := &client{bridgeKey: "B", dataKey: "D"}
	req, _ := http.NewRequest(http.MethodGet, "http://x", nil)
	if err := c.applyAuth(context.Background(), req, authMode(99)); err == nil {
		t.Error("expected error for unknown auth mode")
	}
}
