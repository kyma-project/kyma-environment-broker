package command

import (
	"errors"
	"fmt"

	keb "skr-tester/pkg/keb"
	"skr-tester/pkg/logger"

	"github.com/spf13/cobra"
)

type DeprovisionCommand struct {
	cobraCmd        *cobra.Command
	log             logger.Logger
	shootName       string
	globalAccountID string
	subAccountID    string
	outputPath      string
	instanceID      string
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
	deprovisionResp, err := kebClient.DeprovisionSKR(cmd.instanceID)
	if err != nil {
		fmt.Printf("Error deprovisioning SKR: %v\n", err)
	} else {
		fmt.Printf("Deprovisioning response: %v\n", deprovisionResp)
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

func promptUser(msg string) bool {
	fmt.Printf("%s%s", "? ", msg)
	for {
		fmt.Print("Type [y/N]: ")
		var res string
		if _, err := fmt.Scanf("%s", &res); err != nil {
			return false
		}
		switch res {
		case "yes", "y":
			return true
		case "No", "N", "no", "n":
			return false
		default:
			continue
		}
	}
}
