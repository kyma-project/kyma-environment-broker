package handlers

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
)

type Handler interface {
	AttachRoutes(router *mux.Router)
}

type handler struct {
	handlers []Handler
}

func NewOrchestrationHandler(db storage.BrokerStorage, clusterQueue *process.Queue, defaultMaxPage int, log *slog.Logger) Handler {
	return &handler{
		handlers: []Handler{
			NewKymaHandler(),
			NewClusterHandler(db.Orchestrations(), clusterQueue, log),
			NewOrchestrationStatusHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), clusterQueue, defaultMaxPage, log),
		},
	}
}

func (h *handler) AttachRoutes(router *mux.Router) {
	for _, handler := range h.handlers {
		handler.AttachRoutes(router)
	}
}

func validateTarget(spec orchestration.TargetSpec) error {
	if spec.Include == nil || len(spec.Include) == 0 {
		return errors.New("targets.include array must be not empty")
	}
	return nil
}

// ValidateDeprecatedParameters cheks if `maintenanceWindow` parameter is used as schedule.
func ValidateDeprecatedParameters(params orchestration.Parameters) error {
	if params.Strategy.Schedule == string(orchestration.MaintenanceWindow) {
		return fmt.Errorf("{\"strategy\":{\"schedule\": \"maintenanceWindow\"} is deprecated use {\"strategy\":{\"MaintenanceWindow\": true} instead")
	}
	return nil
}

// ValidateScheduleParameter cheks if the schedule parameter is valid.
func ValidateScheduleParameter(params *orchestration.Parameters) error {
	switch params.Strategy.Schedule {
	case "immediate":
	case "now":
		params.Strategy.ScheduleTime = time.Now()
	default:
		parsedTime, err := time.Parse(time.RFC3339, params.Strategy.Schedule)
		if err == nil {
			params.Strategy.ScheduleTime = parsedTime
		} else {
			return fmt.Errorf("the schedule filed does not contain 'imediate'/'now' nor is a date: %w", err)
		}
	}
	return nil
}
