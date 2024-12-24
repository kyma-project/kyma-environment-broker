package metricsv2

import (
	"fmt"
	"log/slog"
	"sync"
)

var (
	m sync.Mutex
)

func Debug(logger *slog.Logger, source, message string) {
	m.Lock()
	defer m.Unlock()
	logger.Info(fmt.Sprintf("#Debug logs: (%s) : %s", source, message))
}
