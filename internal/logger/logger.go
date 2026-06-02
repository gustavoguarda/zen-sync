// Package logger creates rotating file loggers for the daemon and CLI.
//
// Rotation: 1 MiB per file, keep 3 backups. We pin those defaults here
// rather than parameterizing them because they are products of "what feels
// right for a small helper" and there is no use case yet for tuning them.
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// New returns a *log.Logger that writes to path with size-based rotation.
// The returned closeFn flushes and closes the underlying sink and must be
// called before the process exits.
func New(path string) (*log.Logger, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	lj := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    1, // MiB
		MaxBackups: 3,
		LocalTime:  true,
	}
	lg := log.New(io.Writer(lj), "", log.LstdFlags|log.Lmicroseconds)
	return lg, lj.Close, nil
}
