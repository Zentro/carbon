package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"carbon/domain"
	"carbon/internal/api_key"
	"carbon/internal/api_login_key"
	"carbon/internal/client"
	"carbon/internal/resource"
	"carbon/internal/server"
	"carbon/internal/user"
	"carbon/remote"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// fakeRemote satisfies remote.Client for context-injection tests. Methods
// panic if called unexpectedly so misuse is loud.
type fakeRemote struct{ remote.Client }

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      rawDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		rawDB.Close()
		t.Fatalf("gorm.Open: %v", err)
	}
	return gdb, mock, rawDB
}

func TestAttachCorsHeaders(t *testing.T) {
	r := gin.New()
	r.Use(AttachCorsHeaders())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	cases := map[string]string{
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Origin":      "*",
		"Access-Control-Max-Age":           "7200",
	}
	for k, want := range cases {
		if got := w.Header().Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Access-Control-Allow-Headers should be set")
	}
}

func TestAttachExtract_ApiClient(t *testing.T) {
	want := &fakeRemote{}
	r := gin.New()
	r.Use(AttachApiClient(want))
	var got remote.Client
	r.GET("/x", func(c *gin.Context) {
		got = ExtractApiClient(c)
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if got != want {
		t.Errorf("ExtractApiClient did not return injected client (got %v)", got)
	}
}

func TestExtractApiClient_PanicsWhenAbsent(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	r := gin.New()
	r.GET("/x", func(c *gin.Context) { ExtractApiClient(c) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
}

func TestAttachExtract_AllManagers(t *testing.T) {
	gdb, _, raw := newMockDB(t)
	defer raw.Close()

	rm, _ := resource.NewManager(context.Background(), &fakeResourceRemote{})
	sm, _ := server.NewManager(context.Background(), gdb)
	um, _ := user.NewManager(context.Background(), &fakeRemote{})
	tm, _ := api_login_key.NewManager(context.Background(), gdb)
	km, _ := api_key.NewManager(context.Background(), gdb)
	cm, _ := client.NewManager(context.Background(), gdb)

	r := gin.New()
	r.Use(
		AttachResourceManager(rm),
		AttachServerManager(sm),
		AttachUserManager(um),
		AttachApiLoginKeyManager(tm),
		AttachApiKeyManager(km),
		AttachClientManager(cm),
	)
	r.GET("/x", func(c *gin.Context) {
		if ExtractResourceManager(c) != rm {
			t.Error("ResourceManager mismatch")
		}
		if ExtractServerManager(c) != sm {
			t.Error("ServerManager mismatch")
		}
		if ExtractApiLoginKeyManager(c) != tm {
			t.Error("ApiLoginKeyManager mismatch")
		}
		if ExtractApiKeyManager(c) != km {
			t.Error("ApiKeyManager mismatch")
		}
		if ExtractClientManager(c) != cm {
			t.Error("ClientManager mismatch")
		}
		c.Status(http.StatusOK)
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
}

// fakeResourceRemote returns an empty resource set so NewManager doesn't fail.
type fakeResourceRemote struct{ remote.Client }

func (*fakeResourceRemote) GetResources(context.Context) ([]domain.Resource, error) {
	return nil, nil
}

func TestServerExists_NotFound(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	sm, _ := server.NewManager(context.Background(), gdb)

	id := uuid.New()
	mock.ExpectQuery("SELECT .* FROM `servers`").
		WillReturnError(gorm.ErrRecordNotFound)

	r := gin.New()
	r.Use(AttachServerManager(sm))
	r.GET("/servers/:server", ServerExists(), func(c *gin.Context) {
		t.Error("handler should not be reached")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/servers/"+id.String(), nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestServerExists_FoundSetsContext(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	sm, _ := server.NewManager(context.Background(), gdb)

	id := uuid.New()
	rows := sqlmock.NewRows([]string{"server_id", "name"}).AddRow(id, "n")
	mock.ExpectQuery("SELECT .* FROM `servers`").
		WithArgs(id, 1).
		WillReturnRows(rows)

	r := gin.New()
	r.Use(AttachServerManager(sm))
	r.GET("/servers/:server", ServerExists(), func(c *gin.Context) {
		s := ExtractServer(c)
		if s.ServerID != id {
			t.Errorf("server.ServerID = %s, want %s", s.ServerID, id)
		}
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/servers/"+id.String(), nil))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestResourceExists_NotFound(t *testing.T) {
	rm, _ := resource.NewManager(context.Background(), &fakeResourceRemote{})

	r := gin.New()
	r.Use(AttachResourceManager(rm))
	r.GET("/resources/:resource", ResourceExists(), func(c *gin.Context) {
		t.Error("handler should not be reached")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/resources/99", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestResourceExists_FoundSetsContext(t *testing.T) {
	rm, _ := resource.NewManager(context.Background(), &fakeResourceRemote{})
	rm.Add(&domain.Resource{ResourceId: 7, Title: "t"})

	r := gin.New()
	r.Use(AttachResourceManager(rm))
	r.GET("/resources/:resource", ResourceExists(), func(c *gin.Context) {
		got := ExtractResource(c)
		if got.ResourceId != 7 {
			t.Errorf("ResourceId = %d, want 7", got.ResourceId)
		}
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/resources/7", nil))
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestApiKeyExists_InvalidIntBadRequest(t *testing.T) {
	gdb, _, raw := newMockDB(t)
	defer raw.Close()
	km, _ := api_key.NewManager(context.Background(), gdb)

	r := gin.New()
	r.Use(AttachApiKeyManager(km))
	r.GET("/api-keys/:id", ApiKeyExists(), func(c *gin.Context) {
		t.Error("handler should not run")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api-keys/not-int", nil))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestApiKeyExists_NotFound(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	km, _ := api_key.NewManager(context.Background(), gdb)

	mock.ExpectQuery("SELECT .* FROM `api_keys` WHERE api_key_id = ?").
		WithArgs(99, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	r := gin.New()
	r.Use(AttachApiKeyManager(km))
	r.GET("/api-keys/:id", ApiKeyExists(), func(c *gin.Context) {
		t.Error("handler should not run")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api-keys/99", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestApiKeyExists_FoundSetsContext(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	km, _ := api_key.NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"api_key_id", "user_id", "enabled"}).
		AddRow(42, 7, true)
	mock.ExpectQuery("SELECT .* FROM `api_keys` WHERE api_key_id = ?").
		WithArgs(42, 1).
		WillReturnRows(rows)

	r := gin.New()
	r.Use(AttachApiKeyManager(km))
	r.GET("/api-keys/:id", ApiKeyExists(), func(c *gin.Context) {
		v, _ := c.Get("apiKey")
		k := v.(*domain.ApiKey)
		if k.ApiKeyID != 42 {
			t.Errorf("apiKey.ApiKeyID = %d, want 42", k.ApiKeyID)
		}
		c.JSON(http.StatusOK, gin.H{"id": k.ApiKeyID})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api-keys/"+strconv.Itoa(42), nil))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body map[string]int
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["id"] != 42 {
		t.Errorf("body id = %d, want 42", body["id"])
	}
}

func TestExtractResource_PanicsWhenAbsent(t *testing.T) {
	defer func() { _ = recover() }()
	r := gin.New()
	r.GET("/x", func(c *gin.Context) { ExtractResource(c) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	t.Error("expected panic before reaching here")
}

func TestExtractServer_PanicsWhenAbsent(t *testing.T) {
	defer func() { _ = recover() }()
	r := gin.New()
	r.GET("/x", func(c *gin.Context) { ExtractServer(c) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	t.Error("expected panic before reaching here")
}
