package broker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
)

const (
	logKeyInstanceID  = "instanceID"
	logKeyBindingID   = "bindingID"
	logKeyPlanID      = "planID"
	logKeyOperationID = "operationID"
	logKeyStack       = "stack"
)

type WithPanicRecovery struct {
	delegate domain.ServiceBroker
	logger   *slog.Logger
}

func NewWithPanicRecovery(broker domain.ServiceBroker, logger *slog.Logger) *WithPanicRecovery {
	return &WithPanicRecovery{
		delegate: broker,
		logger:   logger,
	}
}

func (w *WithPanicRecovery) Services(ctx context.Context) ([]domain.Service, error) {
	return w.delegate.Services(ctx)
}

func (w *WithPanicRecovery) Provision(ctx context.Context, instanceID string, details domain.ProvisionDetails, asyncAllowed bool) (spec domain.ProvisionedServiceSpec, err error) {
	logger := w.logger.With(logKeyInstanceID, instanceID, logKeyPlanID, details.PlanID)
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error(fmt.Sprintf("panic recovered during provisioning: %v", r), logKeyStack, stack)
			err = apiresponses.NewFailureResponse(
				fmt.Errorf("internal server error during provisioning: %v", r),
				http.StatusInternalServerError,
				fmt.Sprintf("provisioning failed for instance %s", instanceID),
			)
		}
	}()
	return w.delegate.Provision(ctx, instanceID, details, asyncAllowed)
}

func (w *WithPanicRecovery) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, asyncAllowed bool) (spec domain.DeprovisionServiceSpec, err error) {
	logger := w.logger.With(logKeyInstanceID, instanceID, logKeyPlanID, details.PlanID)
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error(fmt.Sprintf("panic recovered during deprovisioning: %v", r), logKeyStack, stack)
			err = apiresponses.NewFailureResponse(
				fmt.Errorf("internal server error during deprovisioning: %v", r),
				http.StatusInternalServerError,
				fmt.Sprintf("deprovisioning failed for instance %s", instanceID),
			)
		}
	}()
	return w.delegate.Deprovision(ctx, instanceID, details, asyncAllowed)
}

func (w *WithPanicRecovery) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (spec domain.UpdateServiceSpec, err error) {
	logger := w.logger.With(logKeyInstanceID, instanceID, logKeyPlanID, details.PlanID)
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error(fmt.Sprintf("panic recovered during update: %v", r), logKeyStack, stack)
			err = apiresponses.NewFailureResponse(
				fmt.Errorf("internal server error during update: %v", r),
				http.StatusInternalServerError,
				fmt.Sprintf("update failed for instance %s", instanceID),
			)
		}
	}()
	return w.delegate.Update(ctx, instanceID, details, asyncAllowed)
}

func (w *WithPanicRecovery) GetInstance(ctx context.Context, instanceID string, details domain.FetchInstanceDetails) (spec domain.GetInstanceDetailsSpec, err error) {
	logger := w.logger.With(logKeyInstanceID, instanceID)
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error(fmt.Sprintf("panic recovered during get instance: %v", r), logKeyStack, stack)
			err = apiresponses.NewFailureResponse(
				fmt.Errorf("internal server error during get instance: %v", r),
				http.StatusInternalServerError,
				fmt.Sprintf("get instance failed for %s", instanceID),
			)
		}
	}()
	return w.delegate.GetInstance(ctx, instanceID, details)
}

func (w *WithPanicRecovery) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (spec domain.LastOperation, err error) {
	logger := w.logger.With(logKeyInstanceID, instanceID, logKeyOperationID, details.OperationData)
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error(fmt.Sprintf("panic recovered during last operation: %v", r), logKeyStack, stack)
			err = apiresponses.NewFailureResponse(
				fmt.Errorf("internal server error during last operation: %v", r),
				http.StatusInternalServerError,
				fmt.Sprintf("last operation failed for instance %s", instanceID),
			)
		}
	}()
	return w.delegate.LastOperation(ctx, instanceID, details)
}

func (w *WithPanicRecovery) Bind(ctx context.Context, instanceID, bindingID string, details domain.BindDetails, asyncAllowed bool) (spec domain.Binding, err error) {
	logger := w.logger.With(logKeyInstanceID, instanceID, logKeyBindingID, bindingID)
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error(fmt.Sprintf("panic recovered during bind: %v", r), logKeyStack, stack)
			err = apiresponses.NewFailureResponse(
				fmt.Errorf("internal server error during bind: %v", r),
				http.StatusInternalServerError,
				fmt.Sprintf("binding failed for instance %s", instanceID),
			)
		}
	}()
	return w.delegate.Bind(ctx, instanceID, bindingID, details, asyncAllowed)
}

func (w *WithPanicRecovery) Unbind(ctx context.Context, instanceID, bindingID string, details domain.UnbindDetails, asyncAllowed bool) (spec domain.UnbindSpec, err error) {
	logger := w.logger.With(logKeyInstanceID, instanceID, logKeyBindingID, bindingID)
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error(fmt.Sprintf("panic recovered during unbind: %v", r), logKeyStack, stack)
			err = apiresponses.NewFailureResponse(
				fmt.Errorf("internal server error during unbind: %v", r),
				http.StatusInternalServerError,
				fmt.Sprintf("unbind failed for instance %s", instanceID),
			)
		}
	}()
	return w.delegate.Unbind(ctx, instanceID, bindingID, details, asyncAllowed)
}

func (w *WithPanicRecovery) GetBinding(ctx context.Context, instanceID, bindingID string, details domain.FetchBindingDetails) (spec domain.GetBindingSpec, err error) {
	logger := w.logger.With(logKeyInstanceID, instanceID, logKeyBindingID, bindingID)
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error(fmt.Sprintf("panic recovered during get binding: %v", r), logKeyStack, stack)
			err = apiresponses.NewFailureResponse(
				fmt.Errorf("internal server error during get binding: %v", r),
				http.StatusInternalServerError,
				fmt.Sprintf("get binding failed for instance %s", instanceID),
			)
		}
	}()
	return w.delegate.GetBinding(ctx, instanceID, bindingID, details)
}

func (w *WithPanicRecovery) LastBindingOperation(ctx context.Context, instanceID, bindingID string, details domain.PollDetails) (spec domain.LastOperation, err error) {
	logger := w.logger.With(logKeyInstanceID, instanceID, logKeyBindingID, bindingID, logKeyOperationID, details.OperationData)
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logger.Error(fmt.Sprintf("panic recovered during last binding operation: %v", r), logKeyStack, stack)
			err = apiresponses.NewFailureResponse(
				fmt.Errorf("internal server error during last binding operation: %v", r),
				http.StatusInternalServerError,
				fmt.Sprintf("last binding operation failed for instance %s", instanceID),
			)
		}
	}()
	return w.delegate.LastBindingOperation(ctx, instanceID, bindingID, details)
}
