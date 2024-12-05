package setup

import (
	"log/slog"
	"os"
)

func FatalOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func LogOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
	}
}
