package common

// These variables are set via ldflags during build
var (
	// Version is the semantic version from .version file
	Version = "dev"
	// Build is the build timestamp from .version file
	Build = "unknown"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// GetVersion returns the full version string
func GetVersion() string {
	return Version
}

// GetBuild returns the build timestamp
func GetBuild() string {
	return Build
}

// GetGitCommit returns the git commit hash
func GetGitCommit() string {
	return GitCommit
}

// GetFullVersion returns the complete version information
func GetFullVersion() string {
	if Build != "unknown" {
		return Version + "-" + Build
	}
	return Version
}