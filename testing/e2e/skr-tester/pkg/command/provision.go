package command

import (
	"errors"
	"fmt"

	keb "skr-tester/pkg/keb"
	"skr-tester/pkg/logger"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type ProvisionCommand struct {
	cobraCmd        *cobra.Command
	log             logger.Logger
	shootName       string
	globalAccountID string
	subAccountID    string
	outputPath      string
	planID          string
	region          string
}

func NewProvisionCmd() *cobra.Command {
	cmd := ProvisionCommand{}
	cobraCmd := &cobra.Command{
		Use:     "provision",
		Aliases: []string{"p"},
		Short:   "Provisions a Kyma Runtime",
		Long:    "Provisions a Kyma Runtime",
		Example: "skr-tester provision -p 361c511f-f939-4621-b228-d0fb79a1fe15 -r eu-central-1                           Provisions the SKR.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.planID, "planID", "p", "", "PlanID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.region, "region", "r", "", "Region of the specific Kyma Runtime.")

	return cobraCmd
}

func (cmd *ProvisionCommand) Run() error {
	cmd.log = logger.New()
	kebClient := keb.NewKEBClient(keb.NewKEBConfig())
	dummyCreds := map[string]interface{}{
		"clientid":     "dummy_client_id",
		"clientsecret": "dummy_client_secret",
		"smURL":        "dummy_url",
		"url":          "dummy_token_url",
	}
	instanceID := uuid.New().String()
	customParams := map[string]interface{}{
		"plan_id": cmd.planID,
		"region":  cmd.region,
	}
	fmt.Printf("Instance ID: %s\n", instanceID)
	resp, err := kebClient.ProvisionSKR(instanceID, instanceID, nil, dummyCreds, customParams)
	if err != nil {
		fmt.Printf("Error provisioning SKR: %v\n", err)
	} else {
		fmt.Printf("Provisioning response: %v\n", resp)
	}

	return nil
}

func (cmd *ProvisionCommand) Validate() error {
	if cmd.planID != "" || cmd.region != "" {
		return nil
	} else {
		return errors.New("you must specify the planID and region")
	}
}
