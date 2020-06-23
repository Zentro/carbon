package system

import (
	"github.com/docker/docker/pkg/parsers/kernel"
	"runtime"
)

type Information struct {
	Version       string `json:"version"`
	BuildTime     string `json:"build_time"`
	KernelVersion string `json:"kernel_version"`
	Architecture  string `json:"architecture"`
	OS            string `json:"os"`
	CpuCount      int    `json:"cpu_count"`
}

func GetSystemInformation() (*Information, error) {
	k, err := kernel.GetKernelVersion()
	if err != nil {
		return nil, err
	}

	s := &Information{
		Version:       Version,
		BuildTime:     BuildTime,
		KernelVersion: k.String(),
		Architecture:  runtime.GOARCH,
		OS:            runtime.GOOS,
		CpuCount:      runtime.NumCPU(),
	}

	return s, nil
}
