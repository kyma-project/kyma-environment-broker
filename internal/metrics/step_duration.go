package metrics

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"

	"github.com/prometheus/client_golang/prometheus"
)

// StepDurationCollector provides histograms which describes the time of provisioning/update/deprovisioning steps:
// - kcp_keb_v2_provisioning_step_duration_seconds
// - kcp_keb_v2_update_step_duration_seconds
// - kcp_keb_v2_deprovisioning_step_duration_seconds
type StepDurationCollector struct {
	provisioningStepHistogram   *prometheus.HistogramVec
	updateStepHistogram         *prometheus.HistogramVec
	deprovisioningStepHistogram *prometheus.HistogramVec
}

func NewStepDurationCollector() *StepDurationCollector {
	return &StepDurationCollector{
		provisioningStepHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "provisioning_step_duration_seconds",
			Help:      "The time of the provisioning step",
			Buckets: prometheus.ExponentialBuckets(
				0.001, // 1 ms
				2,     // double each time
				20,    // ~10 minutes
			),
		}, []string{"plan_id", "step_name"}),
		updateStepHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "update_step_duration_seconds",
			Help:      "The time of the update step",
			Buckets: prometheus.ExponentialBuckets(
				0.001, // 1 ms
				2,     // double each time
				20,    // ~10 minutes
			),
		}, []string{"plan_id", "step_name"}),
		deprovisioningStepHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "deprovisioning_step_duration_seconds",
			Help:      "The time of the deprovisioning step",
			Buckets: prometheus.ExponentialBuckets(
				0.001, // 1 ms
				2,     // double each time
				20,    // ~10 minutes
			),
		}, []string{"plan_id", "step_name"}),
	}
}

func (c *StepDurationCollector) Describe(ch chan<- *prometheus.Desc) {
	c.provisioningStepHistogram.Describe(ch)
	c.updateStepHistogram.Describe(ch)
	c.deprovisioningStepHistogram.Describe(ch)
}

func (c *StepDurationCollector) Collect(ch chan<- prometheus.Metric) {
	c.provisioningStepHistogram.Collect(ch)
	c.updateStepHistogram.Collect(ch)
	c.deprovisioningStepHistogram.Collect(ch)
}

func (c *StepDurationCollector) OnOperationStepProcessed(_ context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.OperationStepProcessed)
	if !ok {
		return fmt.Errorf("expected process.OperationStepProcessed in OnOperationStepProcessed but got %+v", ev)
	}

	if stepProcessed.StepName == "" {
		return fmt.Errorf("step name is empty for operation ID: %s", stepProcessed.Operation.ID)
	}

	switch stepProcessed.Operation.Type {
	case internal.OperationTypeProvision:
		c.provisioningStepHistogram.
			WithLabelValues(stepProcessed.Operation.ProvisioningParameters.PlanID, stepProcessed.StepName).
			Observe(stepProcessed.Duration.Seconds())
	case internal.OperationTypeUpdate:
		c.updateStepHistogram.
			WithLabelValues(stepProcessed.Operation.ProvisioningParameters.PlanID, stepProcessed.StepName).
			Observe(stepProcessed.Duration.Seconds())
	case internal.OperationTypeDeprovision:
		c.deprovisioningStepHistogram.
			WithLabelValues(stepProcessed.Operation.ProvisioningParameters.PlanID, stepProcessed.StepName).
			Observe(stepProcessed.Duration.Seconds())
	}

	return nil
}
