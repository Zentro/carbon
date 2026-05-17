package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return p
}

func TestFromFile_ParsesValidYAML(t *testing.T) {
	body := `
key: abc
secret: shh
log_directory: /tmp/logs
root_directory: /tmp/root
debug: true
api:
  host: 127.0.0.1
  port: 9090
  read_timeout: 5
  write_timeout: 10
  idle_timeout: 30
db:
  host: db.local
  port: 3307
  database: carbon
  username: u
  password: p
remote:
  location: https://forum.example.org/api
  bridge_key: bk
  data_key: dk
  oauth:
    client_id: cid
    client_secret: csecret
`
	p := writeTempConfig(t, body)
	if err := FromFile(p); err != nil {
		t.Fatalf("FromFile: %v", err)
	}

	c := Get()
	if c.Key != "abc" || c.Secret != "shh" {
		t.Errorf("Key/Secret mismatch: %+v", c)
	}
	if c.Api.Host != "127.0.0.1" || c.Api.Port != 9090 {
		t.Errorf("Api host/port mismatch: %+v", c.Api)
	}
	if c.Api.ReadTimeout != time.Duration(5) {
		t.Errorf("ReadTimeout: got %v want 5", c.Api.ReadTimeout)
	}
	if c.Db.Host != "db.local" || c.Db.Port != 3307 || c.Db.Database != "carbon" {
		t.Errorf("Db mismatch: %+v", c.Db)
	}
	if c.Remote.Location != "https://forum.example.org/api" || c.Remote.BridgeKey != "bk" || c.Remote.DataKey != "dk" {
		t.Errorf("Remote mismatch: %+v", c.Remote)
	}
	if c.Remote.OAuth.ClientID != "cid" || c.Remote.OAuth.ClientSecret != "csecret" {
		t.Errorf("OAuth mismatch: %+v", c.Remote.OAuth)
	}
}

func TestFromFile_MissingPathReturnsError(t *testing.T) {
	if err := FromFile(filepath.Join(t.TempDir(), "nope.yml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFromFile_InvalidYAMLReturnsError(t *testing.T) {
	p := writeTempConfig(t, "not: [valid: yaml")
	if err := FromFile(p); err == nil {
		t.Fatal("expected error parsing invalid yaml")
	}
}

func TestSetAndGet_ReturnsCopy(t *testing.T) {
	original := &Configuration{Key: "k1"}
	Set(original)

	got := Get()
	got.Key = "mutated"

	if Get().Key != "k1" {
		t.Errorf("Get() should return a copy; stored value was mutated: %q", Get().Key)
	}
}

func TestDefaultLocation(t *testing.T) {
	if DefaultLocation != "/etc/carbon/config.yml" {
		t.Errorf("DefaultLocation changed unexpectedly: %q", DefaultLocation)
	}
}
