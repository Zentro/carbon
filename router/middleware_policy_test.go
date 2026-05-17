package router

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"carbon/domain"
	"carbon/internal/policy"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setPrincipal(p domain.Principal) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(principalCtxKey, p)
		c.Next()
	}
}

func TestAttachExtract_Authorizer(t *testing.T) {
	a := policy.New()
	r := gin.New()
	r.Use(AttachAuthorizer(a))
	r.GET("/x", func(c *gin.Context) {
		if ExtractAuthorizer(c) != a {
			t.Error("ExtractAuthorizer returned different instance")
		}
		c.Status(http.StatusOK)
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
}

func TestExtractAuthorizer_PanicsWhenAbsent(t *testing.T) {
	defer func() { _ = recover() }()
	r := gin.New()
	r.GET("/x", func(c *gin.Context) { ExtractAuthorizer(c) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	t.Error("expected panic")
}

func TestRequireCan_CollectionAction_Allowed(t *testing.T) {
	r := gin.New()
	p := domain.Principal{UserID: 1, Scopes: domain.NewScopeSet("*")}
	r.Use(AttachAuthorizer(policy.New()), setPrincipal(p))
	r.GET("/x", RequireCan(domain.ActionServerList, ""), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestRequireCan_InsufficientScope_403(t *testing.T) {
	r := gin.New()
	p := domain.Principal{UserID: 1, Scopes: domain.NewScopeSet()}
	r.Use(AttachAuthorizer(policy.New()), setPrincipal(p))
	r.GET("/x", RequireCan(domain.ActionServerWrite, ""), func(c *gin.Context) {
		t.Error("should not reach handler")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestRequireCan_WithLoadedTarget(t *testing.T) {
	srv := &domain.Server{ServerID: uuid.New(), OwnerUserID: 7}

	r := gin.New()
	p := domain.Principal{UserID: 7, Scopes: domain.RoleUser.Scopes()}
	r.Use(AttachAuthorizer(policy.New()), setPrincipal(p),
		func(c *gin.Context) { c.Set("server", srv); c.Next() })
	r.GET("/x", RequireCan(domain.ActionServerWrite, "server"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRequireCan_NonOwnerLoadedTarget_403(t *testing.T) {
	srv := &domain.Server{ServerID: uuid.New(), OwnerUserID: 99}

	r := gin.New()
	p := domain.Principal{UserID: 7, Scopes: domain.RoleUser.Scopes()}
	r.Use(AttachAuthorizer(policy.New()), setPrincipal(p),
		func(c *gin.Context) { c.Set("server", srv); c.Next() })
	r.GET("/x", RequireCan(domain.ActionServerWrite, "server"), func(c *gin.Context) {
		t.Error("should not reach handler")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestRequireCan_MissingTargetKey_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when loader middleware did not set target key")
		}
	}()
	r := gin.New()
	p := domain.Principal{UserID: 7, Scopes: domain.NewScopeSet("*")}
	r.Use(AttachAuthorizer(policy.New()), setPrincipal(p))
	// "server" never gets set in ctx — RequireCan must panic.
	r.GET("/x", RequireCan(domain.ActionServerWrite, "server"), func(c *gin.Context) {})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
}

func TestRequireCan_TargetWrongType_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when target does not implement Manageable")
		}
	}()
	r := gin.New()
	p := domain.Principal{UserID: 7, Scopes: domain.NewScopeSet("*")}
	r.Use(AttachAuthorizer(policy.New()), setPrincipal(p),
		func(c *gin.Context) { c.Set("server", "not a Manageable"); c.Next() })
	r.GET("/x", RequireCan(domain.ActionServerWrite, "server"), func(c *gin.Context) {})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
}

func TestTranslatePolicyError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"insufficient scope", policy.ErrInsufficientScope, http.StatusForbidden},
		{"forbidden", policy.ErrForbidden, http.StatusForbidden},
		{"kind mismatch is a wiring bug → 500", policy.ErrKindMismatch, http.StatusInternalServerError},
		{"unknown error defaults to 403", errors.New("other"), http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := translatePolicyError(tc.err)
			if status != tc.wantStatus {
				t.Errorf("status = %d, want %d", status, tc.wantStatus)
			}
			if msg == "" {
				t.Error("translatePolicyError should always return a non-empty user message")
			}
		})
	}
}
