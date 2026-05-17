package cmd

import (
	"bytes"
	"carbon/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCmd_Use(t *testing.T) {
	if rootCmd.Use != "carbon" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "carbon")
	}
}

func TestRootCmd_PersistentFlagsRegistered(t *testing.T) {
	for _, name := range []string{"debug", "config", "auto-tls", "tls-hostname"} {
		if rootCmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("expected persistent flag %q to be registered", name)
		}
	}
}

func TestRootCmd_ConfigFlagDefaultsToDefaultLocation(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("config")
	if f == nil {
		t.Fatal("config flag missing")
	}
	if f.DefValue != config.DefaultLocation {
		t.Errorf("config flag default = %q, want %q", f.DefValue, config.DefaultLocation)
	}
}

func TestVersionCmd_OutputsVersion(t *testing.T) {
	var found bool
	for _, c := range rootCmd.Commands() {
		if c.Use == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("version subcommand not registered on rootCmd")
	}

	// Execute the version command and capture stdout.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	versionCmd.Run(versionCmd, nil)
	w.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "v") {
		t.Errorf("expected version output to start with 'v', got %q", out)
	}
	if !strings.Contains(out, "Rafael Galvan") {
		t.Errorf("expected copyright line, got %q", out)
	}
}

func TestInitConfig_LoadsTempFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yml")
	body := "key: from-init\nsecret: s\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	orig := configPath
	configPath = p
	defer func() { configPath = orig }()

	initConfig()

	c := config.Get()
	if c.Key != "from-init" {
		t.Errorf("config.Key = %q, want %q", c.Key, "from-init")
	}
}
