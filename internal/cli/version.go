// Package cli implements zen-sync subcommand handlers.
//
// Each subcommand exposes a top-level function called by the dispatcher in
// cmd/zen-sync. Subcommands accept their own args slice (sans the program
// name and subcommand name) and return an error.
package cli

import (
	"fmt"
	"io"
)

// Version writes the build metadata to w.
// The values are passed in (rather than read from globals) so the test can
// pin them; production callers will use the values from main.go's ldflags.
func Version(w io.Writer, version, commit, buildDate string) error {
	_, err := fmt.Fprintf(w, "zen-sync %s\ncommit:  %s\nbuilt:   %s\n", version, commit, buildDate)
	return err
}
