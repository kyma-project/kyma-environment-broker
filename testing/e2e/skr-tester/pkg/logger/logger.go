package logger

import (
	"os"

	"log/slog"

	"github.com/spf13/pflag"
)

// CfgLevel is the configured logging level
var CfgLevel int

// New returns a Logger with the standard slog.Logger
func New() Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.Level(CfgLevel),
	})
	log := slog.New(handler)
	return &logging{
		Logger:    log,
		verbosity: CfgLevel,
	}
}

// Logger is the interface to interact with a CLI logging instance
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	AddFlags(f *pflag.FlagSet)
	V(level int) Verbose
	IsEnabled(level int) bool
}

// logging provides logging facility using slog.
type logging struct {
	*slog.Logger
	verbosity int
}

// Verbose is a helper type for conditional logging.
type Verbose bool

// V returns a Verbose instance for conditional logging.
func (l *logging) V(level int) Verbose {
	return Verbose(l.verbosity >= level)
}

// IsEnabled checks if the given verbosity level is enabled.
func (l *logging) IsEnabled(level int) bool {
	return l.verbosity >= level
}

// AddFlags adds logging flags to the given flag set.
func (l *logging) AddFlags(f *pflag.FlagSet) {
	f.IntVar(&CfgLevel, "v", 0, "set the log level verbosity")
}

// Info logs an info message.
func (l *logging) Info(msg string, args ...interface{}) {
	l.Logger.Info(msg, args...)
}

// Error logs an error message.
func (l *logging) Error(msg string, args ...interface{}) {
	l.Logger.Error(msg, args...)
}
