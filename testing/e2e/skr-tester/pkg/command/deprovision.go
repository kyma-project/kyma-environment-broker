package command

import (
	"errors"
	"fmt"

	keb "skr-tester/pkg/keb"
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
		Short:   "Deprovisions a Kyma Runtime",
		Long:    "Deprovisions a Kyma Runtime",
		Example: "skr-tester deprovision -i instanceID                            Deprovisions the SKR.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "InstanceID of the specific Kyma Runtime.")

	return cobraCmd
}

func (cmd *DeprovisionCommand) Run() error {
	cmd.log = logger.New()
	kebClient := keb.NewKEBClient(keb.NewKEBConfig())
	resp, err := kebClient.DeprovisionSKR(cmd.instanceID)
	if err != nil {
		fmt.Printf("Error deprovisioning SKR: %v\n", err)
	} else {
		fmt.Printf("Deprovision operationID: %s\n", resp["operation"].(string))
	}

	return nil
}

func (cmd *DeprovisionCommand) Validate() error {
	if cmd.instanceID != "" {
		return nil
	} else {
		return errors.New("at least one of the following options have to be specified: instanceID")
	}
}
