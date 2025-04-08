package version

// Version information
var (
	// Version is the version of the application
	// This is a fallback value that will be overridden during the build process
	// using ldflags to inject the actual version from git tags
	Version = "dev"

	// Commit is the git commit hash
	// This is a fallback value that will be overridden during the build process
	Commit = "unknown"

	// BuildDate is the build date
	// This is a fallback value that will be overridden during the build process
	BuildDate = "unknown"
)
