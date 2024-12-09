package error

import (
	"errors"
	"strings"

	gcli "github.com/kyma-project/kyma-environment-broker/internal/third_party/machinebox/graphql"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apierr2 "k8s.io/apimachinery/pkg/api/meta"
)

const OperationTimeOutMsg string = "operation has reached the time limit"
const NotSet = "not-set"

type Reason string
type Component string

type ErrorReporter interface {
	error
	GetReason() Reason
	GetComponent() Component
}

// error reporter
type LastError struct {
	Message   string    `json:"message,omitempty"`
	Reason    Reason    `json:"reason,omitempty"`
	Component Component `json:"component,omitempty"`
	Step      string    `json:"step,omitempty"`
	Team      string    `json:"team,omitempty"`
}

const (
	KEBInternalCode         Reason = "err_keb_internal"
	KEBTimeOutCode          Reason = "err_keb_timeout"
	ProvisionerCode         Reason = "err_provisioner_nil_last_error"
	HttpStatusCode          Reason = "err_http_status_code"
	ClusterNotFoundCode     Reason = "err_cluster_not_found"
	K8SUnexpectedServerCode Reason = "err_k8s_unexpected_server_error"
	K8SUnexpectedObjectCode Reason = "err_k8s_unexpected_object_error"
	K8SNoMatchCode          Reason = "err_k8s_no_match_error"
	K8SAmbiguousCode        Reason = "err_k8s_ambiguous_error"
)

const (
	KebDbDependency                 Component = "db - keb"
	K8sDependency                   Component = "k8s client - keb"
	KEBDependency                   Component = "keb"
	EDPDependency                   Component = "edp"
	ProvisionerDependency           Component = "provisioner"
	ReconcileDependency             Component = "reconciler"
	InfrastructureManagerDependency Component = "infrastructure-manager"
	LifeCycleManagerDependency      Component = "lifecycle-manager"
	BtpManagerDependency            Component = "btp-manager"
	AccountPoolDependency           Component = "account-pool"
)

func (err LastError) GetReason() Reason {
	return err.Reason
}

func (err LastError) GetComponent() Component {
	return err.Component
}

func (err LastError) Error() string {
	return err.Message
}

func (err LastError) SetComponent(component Component) LastError {
	err.Component = component
	return err
}

func (err LastError) SetReason(reason Reason) LastError {
	err.Reason = reason
	return err
}

func (err LastError) SetMessage(msg string) LastError {
	err.Message = msg
	return err
}

func (err LastError) GetStep() string {
	return err.Step
}

func (err LastError) SetStep(step string) LastError {
	err.Step = step
	return err
}

func TimeoutError(msg, step string, component Component) LastError {
	return LastError{
		Message:   msg,
		Reason:    KEBTimeOutCode,
		Component: component,
		Step:      step,
	}
}

func ReasonForError(err error, step string, component Component) LastError {
	if err == nil {
		return LastError{}
	}

	cause := UnwrapAll(err)

	if lastErr := checkK8SError(cause); lastErr.Component == K8sDependency {
		lastErr.Message = err.Error()
		lastErr.Step = step
		return lastErr
	}

	if status := ErrorReporter(nil); errors.As(cause, &status) {
		return LastError{
			Message:   err.Error(),
			Reason:    status.GetReason(),
			Component: status.GetComponent(),
			Step:      step,
		}
	}

	if ee, ok := cause.(gcli.ExtendedError); ok {
		var errReason Reason
		var errComponent Component
		var errStep string

		reason, found := ee.Extensions()["error_reason"]
		if found {
			if r, ok := reason.(string); ok {
				errReason = Reason(r)
			}
		}
		component, found := ee.Extensions()["error_component"]
		if found {
			if c, ok := component.(string); ok {
				errComponent = Component(c)
			}
		}
		step, found := ee.Extensions()["error_step"]
		if found {
			if s, ok := step.(string); ok {
				errStep = s
			}
		}

		return LastError{
			Message:   err.Error(),
			Reason:    errReason,
			Component: errComponent,
			Step:      errStep,
		}
	}

	if strings.Contains(err.Error(), OperationTimeOutMsg) {
		return TimeoutError(err.Error(), step, component)
	}

	return LastError{
		Message:   err.Error(),
		Reason:    KEBInternalCode,
		Component: component,
		Step:      step,
	}
}

func checkK8SError(cause error) LastError {
	lastErr := LastError{}
	status := apierr.APIStatus(nil)

	switch {
	case errors.As(cause, &status):
		if apierr.IsUnexpectedServerError(cause) {
			lastErr.Reason = K8SUnexpectedServerCode
		} else {
			// reason could be an empty unknown ""
			lastErr.Reason = Reason(apierr.ReasonForError(cause))
		}
		lastErr.Component = K8sDependency
		return lastErr
	case apierr.IsUnexpectedObjectError(cause):
		lastErr.Reason = K8SUnexpectedObjectCode
	case apierr2.IsAmbiguousError(cause):
		lastErr.Reason = K8SAmbiguousCode
	case apierr2.IsNoMatchError(cause):
		lastErr.Reason = K8SNoMatchCode
	}

	if lastErr.Reason != "" {
		lastErr.Component = K8sDependency
	}

	return lastErr
}

// UnwrapOnce accesses the direct cause of the error if any, otherwise
// returns nil.
func UnwrapOnce(err error) (cause error) {
	switch e := err.(type) {
	case interface{ Unwrap() error }:
		return e.Unwrap()
	}
	return nil
}

// UnwrapAll accesses the root cause object of the error.
// If the error has no cause (leaf error), it is returned directly.
// this is a replacement for github.com/pkg/errors.Cause
func UnwrapAll(err error) error {
	for {
		if cause := UnwrapOnce(err); cause != nil {
			err = cause
			continue
		}
		break
	}
	return err
}
