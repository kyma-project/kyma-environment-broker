package command

import (
	"fmt"

	broker "skr-tester/pkg/broker"
	"skr-tester/pkg/logger"

	"github.com/spf13/cobra"
)

type DeprovisionCommand struct {
	cobraCmd   *cobra.Command
	log        logger.Logger
	instanceID string
}

func NewDeprovisionCmd() *cobra.Command {
	cmd := DeprovisionCommand{}
	cobraCmd := &cobra.Command{
		Use:     "deprovision",
		Aliases: []string{"d"},
		Short:   "Deprovisions an instance",
		Long:    "Deprovisions an instance",
		Example: "	skr-tester deprovision -i instanceID                            Deprovisions the instance.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "Instance ID of the specific instance.")

	return cobraCmd
}

func (cmd *DeprovisionCommand) Run() error {
	cmd.log = logger.New()
	brokerClient := broker.NewBrokerClient(broker.NewBrokerConfig())
	resp, _, err := brokerClient.DeprovisionInstance(cmd.instanceID)
	if err != nil {
		fmt.Printf("Error deprovisioning instance: %v\n", err)
	} else {
		fmt.Printf("Deprovision operationID: %s\n", resp["operation"].(string))
	}

	return nil
}

func (cmd *DeprovisionCommand) Validate() error {
	if cmd.instanceID != "" {
		return nil
	} else {
		return fmt.Errorf("at least one of the following options have to be specified: instanceID")
	}
}
