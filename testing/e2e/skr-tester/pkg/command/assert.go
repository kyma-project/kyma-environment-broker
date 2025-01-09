package command

import (
	"errors"
	"fmt"

	kcp "skr-tester/pkg/kcp"
	"skr-tester/pkg/logger"

	"github.com/spf13/cobra"
)

type AssertCommand struct {
	cobraCmd    *cobra.Command
	log         logger.Logger
	instanceID  string
	machineType string
	OIDC        string
}

func NewAsertCmd() *cobra.Command {
	cmd := AssertCommand{}
	cobraCmd := &cobra.Command{
		Use:     "assert",
		Aliases: []string{"a"},
		Short:   "Does an assertion",
		Long:    "Does an assertion",
		Example: "skr-tester assert -i instanceID -m m6i.large                         Asserts the instance has the machine type m6i.large.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "InstanceID of the specific instance.")
	cobraCmd.Flags().StringVarP(&cmd.machineType, "machineType", "m", "", "MachineType of the specific instance.")
	cobraCmd.Flags().StringVarP(&cmd.OIDC, "OIDC", "o", "", "OIDC of the specific instance.")

	return cobraCmd
}

func (cmd *AssertCommand) Run() error {
	cmd.log = logger.New()
	kcpClient, err := kcp.NewKCPClient()
	if err != nil {
		return fmt.Errorf("failed to create KCP client: %v", err)
	}
	if cmd.machineType != "" {
		currentMachineType, err := kcpClient.GetCurrentMachineType(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get current machine type: %v", err)
		}
		if cmd.machineType != *currentMachineType {
			return fmt.Errorf("machine types are not equal: expected %s, got %s", cmd.machineType, *currentMachineType)
		} else {
			fmt.Println("Machine type assertion passed: expected and got", cmd.machineType)
		}
	} else if cmd.OIDC != "" {
		currentOIDC, err := kcpClient.GetCurrentOIDCConfig(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get current OIDC: %v", err)
		}
		if cmd.OIDC != fmt.Sprintf("%v", currentOIDC) {
			return fmt.Errorf("OIDCs are not equal: expected %s, got %s", cmd.OIDC, fmt.Sprintf("%v", currentOIDC))
		} else {
			fmt.Println("OIDC assertion passed: expected and got", cmd.OIDC)
		}
	}
	return nil
}

func (cmd *AssertCommand) Validate() error {
	if cmd.instanceID == "" {
		return errors.New("instanceID must be specified")
	}
	if cmd.machineType == "" && cmd.OIDC == "" {
		return errors.New("either machineType or OIDC must be specified")
	}
	if cmd.machineType != "" && cmd.OIDC != "" {
		return errors.New("only one of machineType or OIDC must be specified")
	}
	return nil
}
