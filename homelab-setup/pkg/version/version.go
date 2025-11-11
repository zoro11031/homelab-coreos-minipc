package version

import "fmt"

var (
	// Version is the current version of the application
	Version = "0.1.0-dev"

	// GitCommit is the git commit hash (set during build)
	GitCommit = "unknown"

	// BuildDate is the build date (set during build)
	BuildDate = "unknown"
)

// Info returns formatted version information
func Info() string {
	return fmt.Sprintf("homelab-setup version %s (commit: %s, built: %s)",
		Version, GitCommit, BuildDate)
}

// Short returns just the version number
func Short() string {
	return Version
}
