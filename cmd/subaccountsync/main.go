package main

import (
	"fmt"
	"log/slog"
	"os"
)

const (
	AppPrefix = "subaccount-sync"
)

func main() {
	// create slog logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info(fmt.Sprintf("%s app stub started", AppPrefix))

}
