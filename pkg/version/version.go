package version

import (
	"runtime"

	"github.com/shirou/gopsutil/v3/host"
)

var (
	gitCommit string
	version   string
	buildDate string
)

// Version holds version data
type Version struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoLang    string `json:"goLang"`
	Platform  string `json:"platform"`
	Arch      string `json:"arch"`
	HostID    string `json:"hostId"`
}

// Get returns the Version object
func Get() Version {
	hostId, err := host.HostID()
	if err != nil {
		hostId = err.Error()
	}
	return Version{
		GitCommit: gitCommit,
		Version:   version,
		BuildDate: buildDate,
		GoLang:    runtime.Version(),
		Platform:  runtime.GOOS,
		Arch:      runtime.GOARCH,
		HostID:    hostId,
	}
}
