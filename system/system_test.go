package system

import (
	"runtime"
	"testing"
)

func TestVersionDefaults(t *testing.T) {
	if Version == "" {
		t.Error("Version should have a default value (linker overrides at build time)")
	}
	if BuildTime == "" {
		t.Error("BuildTime should have a default value")
	}
}

func TestGetSystemInformation(t *testing.T) {
	info, err := GetSystemInformation()
	if err != nil {
		t.Fatalf("GetSystemInformation: %v", err)
	}
	if info == nil {
		t.Fatal("GetSystemInformation returned nil info")
	}

	if info.Version != Version {
		t.Errorf("Version = %q, want %q", info.Version, Version)
	}
	if info.BuildTime != BuildTime {
		t.Errorf("BuildTime = %q, want %q", info.BuildTime, BuildTime)
	}
	if info.Architecture != runtime.GOARCH {
		t.Errorf("Architecture = %q, want %q", info.Architecture, runtime.GOARCH)
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.CpuCount != runtime.NumCPU() {
		t.Errorf("CpuCount = %d, want %d", info.CpuCount, runtime.NumCPU())
	}
	if info.KernelVersion == "" {
		t.Error("KernelVersion should not be empty on Linux test runners")
	}
}
