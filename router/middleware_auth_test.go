package router

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"carbon/domain"
	"carbon/internal/api_key"
	"carbon/internal/api_login_key"
	"carbon/remote"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// stubRemote returns a canned User from GetUser; everything else panics.
type stubRemote struct {
	remote.Client
	user    domain.User
	userErr error
}

func (s *stubRemote) GetUser(context.Context, int) (domain.User, error) {
	return s.user, s.userErr
}

func mkLoginContext(t *testing.T) (*gin.Engine, sqlmock.Sqlmock, *sql.DB, *api_login_key.Manager) {
	t.Helper()
	gdb, mock, raw := newMockDB(t)
	mgr, _ := api_login_key.NewManager(context.Background(), gdb)
	return gin.New(), mock, raw, mgr
}

func mkApiKeyContext(t *testing.T) (*gin.Engine, sqlmock.Sqlmock, *sql.DB, *api_key.Manager) {
	t.Helper()
	gdb, mock, raw := newMockDB(t)
	mgr, _ := api_key.NewManager(context.Background(), gdb)
	return gin.New(), mock, raw, mgr
}

// ---- bearerFrom ----

func TestBearerFrom(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Request.Header.Set("Authorization", "Bearer abc123")

	tok, ok := bearerFrom(c, "Authorization")
	if !ok || tok != "abc123" {
		t.Errorf("bearerFrom = (%q, %v), want (\"abc123\", true)", tok, ok)
	}
}

func TestBearerFrom_Malformed(t *testing.T) {
	cases := map[string]string{
		"empty":        "",
		"no scheme":    "abc",
		"wrong scheme": "Basic abc",
		"empty token":  "Bearer ",
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
			if header != "" {
				c.Request.Header.Set("Authorization", header)
			}
			tok, ok := bearerFrom(c, "Authorization")
			if ok || tok != "" {
				t.Errorf("expected (\"\", false) for header %q, got (%q, %v)", header, tok, ok)
			}
		})
	}
}

// ---- RequireAuth ----

func TestRequireAuth_NoCredentials_401(t *testing.T) {
	r := gin.New()
	r.GET("/x", RequireAuth(), func(c *gin.Context) { t.Error("should not reach handler") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("missing WWW-Authenticate: Bearer header")
	}
}

func TestRequireAuth_LoginToken_HappyPath(t *testing.T) {
	r, mock, raw, mgr := mkLoginContext(t)
	defer raw.Close()

	// Token lookup returns a valid row.
	rows := sqlmock.NewRows([]string{
		"api_login_key_id", "user_id", "login_key", "login_key_expires_at", "refresh_key_expires_at",
	}).AddRow(1, 42, "tok", time.Now().Add(time.Hour), time.Now().Add(24*time.Hour))
	mock.ExpectQuery("SELECT .* FROM `api_login_keys`").
		WithArgs("tok", 1).
		WillReturnRows(rows)

	remoteStub := &stubRemote{user: domain.User{UserID: 42, IsStaff: true}}
	r.Use(AttachApiLoginKeyManager(mgr), AttachApiClient(remoteStub))
	r.GET("/x", RequireAuth(), func(c *gin.Context) {
		p := ExtractPrincipal(c)
		if p.UserID != 42 {
			t.Errorf("principal.UserID = %d, want 42", p.UserID)
		}
		if p.CredentialKind != domain.CredentialLogin {
			t.Errorf("CredentialKind = %q, want login", p.CredentialKind)
		}
		if !p.IsAdmin() {
			t.Error("staff user should be admin")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRequireAuth_LoginToken_NotFound_403(t *testing.T) {
	r, mock, raw, mgr := mkLoginContext(t)
	defer raw.Close()

	mock.ExpectQuery("SELECT .* FROM `api_login_keys`").
		WillReturnError(errors.New("not found"))

	r.Use(AttachApiLoginKeyManager(mgr), AttachApiClient(&stubRemote{}))
	r.GET("/x", RequireAuth(), func(c *gin.Context) { t.Error("should not reach handler") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestRequireAuth_LoginToken_Expired_403(t *testing.T) {
	r, mock, raw, mgr := mkLoginContext(t)
	defer raw.Close()

	rows := sqlmock.NewRows([]string{
		"api_login_key_id", "user_id", "login_key", "login_key_expires_at", "refresh_key_expires_at",
	}).AddRow(1, 42, "tok", time.Now().Add(-time.Hour), time.Now().Add(-time.Minute))
	mock.ExpectQuery("SELECT .* FROM `api_login_keys`").
		WithArgs("tok", 1).
		WillReturnRows(rows)

	r.Use(AttachApiLoginKeyManager(mgr), AttachApiClient(&stubRemote{}))
	r.GET("/x", RequireAuth(), func(c *gin.Context) { t.Error("should not reach handler") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestRequireAuth_ApiKey_HappyPath_IntersectsScopes(t *testing.T) {
	r, mock, raw, mgr := mkApiKeyContext(t)
	defer raw.Close()

	rows := sqlmock.NewRows([]string{
		"api_key_id", "user_id", "key", "scopes", "enabled",
	}).AddRow(7, 42, "secret", "server:read", true)
	mock.ExpectQuery("SELECT .* FROM `api_keys`").
		WithArgs("secret", 1).
		WillReturnRows(rows)

	remoteStub := &stubRemote{user: domain.User{UserID: 42, IsStaff: false}}
	r.Use(AttachApiKeyManager(mgr), AttachApiClient(remoteStub))
	r.GET("/x", RequireAuth(), func(c *gin.Context) {
		p := ExtractPrincipal(c)
		if p.CredentialKind != domain.CredentialApi {
			t.Errorf("CredentialKind = %q, want api", p.CredentialKind)
		}
		if !p.Scopes.HasAction(domain.ActionServerRead) {
			t.Error("principal should have server:read from intersected scopes")
		}
		if p.Scopes.HasAction(domain.ActionServerWrite) {
			t.Error("server:write should be filtered out by key.Scopes intersection")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Api-Authorization", "Bearer secret")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRequireAuth_ApiKey_Disabled_403(t *testing.T) {
	r, mock, raw, mgr := mkApiKeyContext(t)
	defer raw.Close()

	rows := sqlmock.NewRows([]string{"api_key_id", "user_id", "key", "enabled"}).
		AddRow(7, 42, "secret", false)
	mock.ExpectQuery("SELECT .* FROM `api_keys`").
		WithArgs("secret", 1).
		WillReturnRows(rows)

	r.Use(AttachApiKeyManager(mgr), AttachApiClient(&stubRemote{}))
	r.GET("/x", RequireAuth(), func(c *gin.Context) { t.Error("should not reach handler") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Api-Authorization", "Bearer secret")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (disabled key)", w.Code)
	}
}

func TestRequireAuth_ApiKey_BoundCarriedThrough(t *testing.T) {
	r, mock, raw, mgr := mkApiKeyContext(t)
	defer raw.Close()

	rows := sqlmock.NewRows([]string{
		"api_key_id", "user_id", "key", "scopes", "enabled", "target_kind", "target_id",
	}).AddRow(7, 42, "secret", "server:read,server:write", true, "server", "uuid-here")
	mock.ExpectQuery("SELECT .* FROM `api_keys`").
		WithArgs("secret", 1).
		WillReturnRows(rows)

	r.Use(AttachApiKeyManager(mgr), AttachApiClient(&stubRemote{user: domain.User{UserID: 42}}))
	r.GET("/x", RequireAuth(), func(c *gin.Context) {
		p := ExtractPrincipal(c)
		if !p.IsBound() {
			t.Error("principal should be bound")
		}
		if p.BoundKind != "server" || p.BoundID != "uuid-here" {
			t.Errorf("binding = (%q,%q), want (server,uuid-here)", p.BoundKind, p.BoundID)
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Api-Authorization", "Bearer secret")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestExtractPrincipal_PanicsWhenAbsent(t *testing.T) {
	defer func() { _ = recover() }()
	r := gin.New()
	r.GET("/x", func(c *gin.Context) { ExtractPrincipal(c) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	t.Error("expected panic")
}
