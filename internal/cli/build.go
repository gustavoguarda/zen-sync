package cli

// Build metadata, set from main.go's ldflags-injected vars at startup.
// Package-level vars (vs. plumbing 3 strings through every subcommand)
// because almost every CLI handler needs at least one of them.
var (
	BuildVersion = "dev"
	BuildCommit  = "none"
	BuildDate    = "unknown"
)
