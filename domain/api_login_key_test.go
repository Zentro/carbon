package domain

import (
	"carbon/config"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func setTestSecret(t *testing.T, secret string) {
	t.Helper()
	config.Set(&config.Configuration{Secret: secret})
}

func TestNewJwtKey_SignsAndParses(t *testing.T) {
	setTestSecret(t, "test-secret")
	exp := time.Now().Add(1 * time.Hour)

	tok, err := NewJwtKey(42, exp)
	if err != nil {
		t.Fatalf("NewJwtKey: %v", err)
	}
	if tok == "" {
		t.Fatal("expected non-empty token")
	}

	claims := &ApiLoginClaims{}
	parsed, err := jwt.ParseWithClaims(tok, claims, func(*jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("token reported invalid")
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(exp.Truncate(time.Second)) {
		t.Errorf("ExpiresAt mismatch: %v vs %v", claims.ExpiresAt, exp)
	}
}

func TestNewJwtKey_WrongSecretFailsValidation(t *testing.T) {
	setTestSecret(t, "right")
	tok, err := NewJwtKey(1, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("NewJwtKey: %v", err)
	}
	_, err = jwt.ParseWithClaims(tok, &ApiLoginClaims{}, func(*jwt.Token) (interface{}, error) {
		return []byte("wrong"), nil
	})
	if err == nil {
		t.Fatal("expected signature validation error")
	}
}

func TestNewApiLoginKey_PopulatesFields(t *testing.T) {
	setTestSecret(t, "s")
	before := time.Now()

	k, err := NewApiLoginKey(7, "10.0.0.1")
	if err != nil {
		t.Fatalf("NewApiLoginKey: %v", err)
	}

	if k.UserID != 7 {
		t.Errorf("UserID = %d, want 7", k.UserID)
	}
	if k.IP != "10.0.0.1" {
		t.Errorf("IP = %q, want 10.0.0.1", k.IP)
	}
	if k.LoginKey == "" || k.RefreshKey == "" {
		t.Error("login/refresh keys should be populated")
	}
	if k.LoginKey == k.RefreshKey {
		t.Error("login key and refresh key should not be identical")
	}

	// LoginKey expires ~24h from now; RefreshKey ~7d from now.
	loginDelta := k.LoginKeyExpiresAt.Sub(before)
	if loginDelta < 23*time.Hour || loginDelta > 25*time.Hour {
		t.Errorf("LoginKeyExpiresAt delta out of range: %v", loginDelta)
	}
	refreshDelta := k.RefreshKeyExpiresAt.Sub(before)
	if refreshDelta < 6*24*time.Hour || refreshDelta > 8*24*time.Hour {
		t.Errorf("RefreshKeyExpiresAt delta out of range: %v", refreshDelta)
	}
	if !k.RefreshKeyExpiresAt.After(k.LoginKeyExpiresAt) {
		t.Error("RefreshKeyExpiresAt should be after LoginKeyExpiresAt")
	}
}
