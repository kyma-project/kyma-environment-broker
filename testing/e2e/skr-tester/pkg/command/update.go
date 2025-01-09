package command

import (
	"errors"
	"fmt"
	broker "skr-tester/pkg/broker"
	kcp "skr-tester/pkg/kcp"
	"skr-tester/pkg/logger"

	"github.com/spf13/cobra"
)

type UpdateCommand struct {
	cobraCmd          *cobra.Command
	log               logger.Logger
	instanceID        string
	planID            string
	updateMachineType bool
	// TODO
	updateOIDC bool
}

func NewUpdateCommand() *cobra.Command {
	cmd := UpdateCommand{}
	cobraCmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"u"},
		Short:   "Update the instnace",
		Long:    "Update the instnace",
		Example: "skr-tester update -i instanceID -p planID --updateMachineType                            Update the instance with new machineType.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "InstanceID of the specific instance.")
	cobraCmd.Flags().StringVarP(&cmd.planID, "planID", "p", "", "PlanID of the specific instance.")
	cobraCmd.Flags().BoolVarP(&cmd.updateMachineType, "updateMachineType", "m", false, "Should update machineType.")

	return cobraCmd
}

func (cmd *UpdateCommand) Run() error {
	cmd.log = logger.New()
	brokerClient := broker.NewBrokerClient(broker.NewBrokerConfig())
	catalog, err := brokerClient.GetCatalog()
	if err != nil {
		return fmt.Errorf("failed to get catalog: %v", err)
	}
	services, ok := catalog["services"].([]interface{})
	if !ok {
		return errors.New("services field not found or invalid in catalog")
	}
	for _, service := range services {
		serviceMap, ok := service.(map[string]interface{})
		if !ok {
			return errors.New("service is not a map[string]interface{}")
		}
		if serviceMap["id"] != broker.KymaServiceID {
			continue
		}
		if cmd.updateMachineType {
			kcpClient, err := kcp.NewKCPClient()
			if err != nil {
				return fmt.Errorf("failed to create KCP client: %v", err)
			}
			currentMachineType, err := kcpClient.GetCurrentMachineType(cmd.instanceID)
			if err != nil {
				return fmt.Errorf("failed to get current machine type: %v", err)
			}
			fmt.Printf("Current machine type: %s\n", *currentMachineType)
			plans, ok := serviceMap["plans"].([]interface{})
			if !ok {
				return errors.New("plans field not found or invalid in serviceMap")
			}
			for _, p := range plans {
				planMap, ok := p.(map[string]interface{})
				if !ok || planMap["id"] != cmd.planID {
					continue
				}
				supportedMachineTypes, err := extractSupportedMachineTypes(planMap)
				if err != nil {
					return fmt.Errorf("failed to extract supportedMachineTypes: %v", err)
				}
				if len(supportedMachineTypes) < 2 {
					continue
				}
				for i, m := range supportedMachineTypes {
					if m == *currentMachineType {
						newMachineType := supportedMachineTypes[(i+1)%len(supportedMachineTypes)].(string)
						fmt.Printf("Determined machine type to update: %s\n", newMachineType)
						resp, err := brokerClient.UpdateInstance(cmd.instanceID, map[string]interface{}{"machineType": newMachineType})
						if err != nil {
							return fmt.Errorf("error updating instance: %v", err)
						}
						fmt.Printf("Update operationID: %s\n", resp["operation"].(string))
						break
					}
				}
			}
		}
	}
	return nil
}

func (cmd *UpdateCommand) Validate() error {
	if cmd.instanceID != "" && cmd.planID != "" {
		return nil
	} else {
		return errors.New("you must specify the planID and instanceID")
	}
}

func extractSupportedMachineTypes(planMap map[string]interface{}) ([]interface{}, error) {
	schemas, ok := planMap["schemas"].(map[string]interface{})
	if !ok {
		return nil, errors.New("schemas field not found or invalid in planMap")
	}
	serviceInstance, ok := schemas["service_instance"].(map[string]interface{})
	if !ok {
		return nil, errors.New("service_instance field not found or invalid in schemas")
	}
	update, ok := serviceInstance["update"].(map[string]interface{})
	if !ok {
		return nil, errors.New("update field not found or invalid in service_instance")
	}
	parameters, ok := update["parameters"].(map[string]interface{})
	if !ok {
		return nil, errors.New("parameters field not found or invalid in update")
	}
	properties, ok := parameters["properties"].(map[string]interface{})
	if !ok {
		return nil, errors.New("properties field not found or invalid in parameters")
	}
	machineType, ok := properties["machineType"].(map[string]interface{})
	if !ok {
		return nil, errors.New("machineType field not found or invalid in properties")
	}
	supportedMachineTypes, ok := machineType["enum"].([]interface{})
	if !ok {
		return nil, errors.New("enum field not found or invalid in machineType")
	}
	return supportedMachineTypes, nil
}
