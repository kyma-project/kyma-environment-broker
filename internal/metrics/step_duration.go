package metrics

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"

	"github.com/prometheus/client_golang/prometheus"
)

// StepDurationCollector provides histograms which describes the time of provisioning steps:
// - kcp_keb_provisioning_step_duration_seconds
type StepDurationCollector struct {
	provisioningStepHistogram *prometheus.HistogramVec
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
				15,
			),
		}, []string{"plan_id", "step_name"}),
	}
}

func (c *StepDurationCollector) Describe(ch chan<- *prometheus.Desc) {
	c.provisioningStepHistogram.Describe(ch)
}

func (c *StepDurationCollector) Collect(ch chan<- prometheus.Metric) {
	c.provisioningStepHistogram.Collect(ch)
}

func (c *StepDurationCollector) OnOperationStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.OperationStepProcessed)
	if !ok {
		return fmt.Errorf("expected process.OperationStepProcessed in OnOperationStepProcessed but got %+v", ev)
	}

	if stepProcessed.StepName == "" {
		return fmt.Errorf("step name is empty for operation ID: %s", stepProcessed.Operation.ID)
	}

	if stepProcessed.Operation.Type == internal.OperationTypeProvision {
		c.provisioningStepHistogram.
			WithLabelValues(stepProcessed.Operation.ProvisioningParameters.PlanID, stepProcessed.StepName).
			Observe(stepProcessed.Duration.Seconds())
	}

	return nil
}
