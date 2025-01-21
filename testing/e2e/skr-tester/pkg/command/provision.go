package command

import (
	"errors"
	"fmt"
	"strings"

	broker "skr-tester/pkg/broker"
	"skr-tester/pkg/logger"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type ProvisionCommand struct {
	cobraCmd      *cobra.Command
	log           logger.Logger
	planID        string
	region        string
	overlapIP     bool
	invalidIP     bool
	validCustomIP bool
}

func NewProvisionCmd() *cobra.Command {
	cmd := ProvisionCommand{}
	cobraCmd := &cobra.Command{
		Use:     "provision",
		Aliases: []string{"p"},
		Short:   "Provisions an instance",
		Long:    "Provisions an instance",
		Example: "skr-tester provision -p 361c511f-f939-4621-b228-d0fb79a1fe15 -r eu-central-1                           Provisions the instance.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.planID, "planID", "p", "", "PlanID of the specific instance.")
	cobraCmd.Flags().StringVarP(&cmd.region, "region", "r", "", "Region of the specific instance.")
	cobraCmd.Flags().BoolVarP(&cmd.overlapIP, "overlapIP", "o", false, "Try to provision with overlapping restricted IP range.")
	cobraCmd.Flags().BoolVarP(&cmd.invalidIP, "invalidIP", "i", false, "Try to provision with invalid IP range.")
	cobraCmd.Flags().BoolVarP(&cmd.validCustomIP, "validCustomIP", "v", false, "Try to provision with valid custom IP range.")

	return cobraCmd
}

func (cmd *ProvisionCommand) Run() error {
	cmd.log = logger.New()
	brokerClient := broker.NewBrokerClient(broker.NewBrokerConfig())
	dummyCreds := map[string]interface{}{
		"clientid":     "dummy_client_id",
		"clientsecret": "dummy_client_secret",
		"smURL":        "dummy_url",
		"url":          "dummy_token_url",
	}
	customParams := map[string]interface{}{}
	if cmd.overlapIP {
		customParams = map[string]interface{}{
			"networking": map[string]interface{}{
				"nodes": "10.242.0.0/22",
			},
		}
	} else if cmd.invalidIP {
		customParams = map[string]interface{}{
			"networking": map[string]interface{}{
				"nodes": "333.242.0.0/22",
			},
		}
	} else if cmd.validCustomIP {
		customParams = map[string]interface{}{
			"networking": map[string]interface{}{
				"nodes": "10.253.0.0/21",
			},
		}
	}
	instanceID := uuid.New().String()
	fmt.Printf("Instance ID: %s\n", instanceID)
	resp, _, err := brokerClient.ProvisionInstance(instanceID, cmd.planID, cmd.region, dummyCreds, customParams)
	if err != nil {
		if cmd.overlapIP && strings.Contains(fmt.Sprintf("%v", resp), "overlap") && strings.Contains(fmt.Sprintf("%v", err), "400") {
			fmt.Println("Provisioning failed due to overlapping IP range, which was expected.")
			return nil
		} else if cmd.invalidIP && strings.Contains(fmt.Sprintf("%v", resp), "invalid CIDR address") && strings.Contains(fmt.Sprintf("%v", err), "400") {
			fmt.Println("Provisioning failed due to invalid CIDR address, which was expected.")
			return nil
		}
		return err
	}
	fmt.Printf("Provision operationID: %s\n", resp["operation"].(string))
	return nil
}

func (cmd *ProvisionCommand) Validate() error {
	if cmd.planID == "" || cmd.region == "" {
		return errors.New("you must specify the planID and region")
	}
	if cmd.overlapIP && cmd.invalidIP {
		return errors.New("you can only set one of overlapIP or invalidIP")
	}
	return nil
}
