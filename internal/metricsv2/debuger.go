package metricsv2

import (
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	m sync.Mutex
)

func Debug(logger logrus.FieldLogger, source, message string) {
	m.Lock()
	defer m.Unlock()
	logger.Infof("#Debug logs: (%s) : %s", source, message)
}
