package main

import (
	"github.com/kyma-project/kyma-environment-broker/common/setup"
)

func main() {
	builder := setup.NewAppBuilder()

	builder.WithConfig()
	builder.WithGardenerClient()
	builder.WithBrokerClient()
	builder.WithStorage()
	builder.WithK8sClient()

	defer builder.Cleanup()

	job := builder.Create()

	err := job.Run()

	if err != nil {
		setup.FatalOnError(err)
	}
}
