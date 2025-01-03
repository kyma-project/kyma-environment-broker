package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	broker "skr-tester/pkg/broker"
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
		Example: "skr-tester update -i instanceID --updateMachineType                            Update the instance with new machineType.",

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
	catalog, _ := brokerClient.GetCatalog()
	services, ok := catalog["services"]
	if !ok {
		return errors.New("state field not found in operation response")
	}
	for _, service := range services.([]interface{}) {
		serviceMap, ok := service.(map[string]interface{})
		if !ok {
			return errors.New("service is not a map[string]interface{}")
		}
		if cmd.updateMachineType {
			currentMachineType, err := getCurrentMachineType(cmd.instanceID)
			if err != nil {
				return fmt.Errorf("failed to get current machine type: %v", err)
			}
			fmt.Printf("Current machine type: %s\n", *currentMachineType)
			if serviceMap["id"] != "47c9dcbf-ff30-448e-ab36-d3bad66ba281" {
				continue
			}
			plans, ok := serviceMap["plans"]
			if !ok {
				return errors.New("plans field not found in serviceMap")
			}
			for _, p := range plans.([]interface{}) {
				planMap := p.(map[string]interface{})
				if planMap["id"] != cmd.planID {
					continue
				}
				updateParams, err := extractUpdateParams(planMap)
				if err != nil {
					return err
				}
				if len(updateParams) < 2 {
					continue
				}
				for i, m := range updateParams {
					if m == *currentMachineType {
						newMachineType := updateParams[(i+1)%len(updateParams)].(string)
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

func getCurrentMachineType(instanceID string) (*string, error) {
	_, err := exec.Command("kcp", "login").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run kcp login: %v", err)
	}
	output, err := exec.Command("kcp", "rt", "-i", instanceID, "-ojson", "--runtime-config").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run kcp rt: %v", err)
	}
	var outputData map[string]interface{}
	if err := json.Unmarshal(output, &outputData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %v", err)
	}
	data, ok := outputData["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, errors.New("data field not found or empty in output")
	}
	runtimeConfig, ok := data[0].(map[string]interface{})["runtimeConfig"].(map[string]interface{})
	if !ok {
		return nil, errors.New("runtimeConfig field not found in data")
	}
	spec, ok := runtimeConfig["spec"].(map[string]interface{})
	if !ok {
		return nil, errors.New("spec field not found in runtimeConfig")
	}
	shoot, ok := spec["shoot"].(map[string]interface{})
	if !ok {
		return nil, errors.New("shoot field not found in spec")
	}
	provider, ok := shoot["provider"].(map[string]interface{})
	if !ok {
		return nil, errors.New("provider field not found in shoot")
	}
	workers, ok := provider["workers"].([]interface{})
	if !ok || len(workers) == 0 {
		return nil, errors.New("workers field not found or empty in provider")
	}
	worker, ok := workers[0].(map[string]interface{})
	if !ok {
		return nil, errors.New("worker field not found in workers")
	}
	machine, ok := worker["machine"].(map[string]interface{})
	if !ok {
		return nil, errors.New("machine field not found in worker")
	}
	machineType, ok := machine["type"].(string)
	if !ok {
		return nil, errors.New("type field not found in machine")
	}
	return &machineType, nil
}

func extractUpdateParams(planMap map[string]interface{}) ([]interface{}, error) {
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
	updateParams, ok := machineType["enum"].([]interface{})
	if !ok {
		return nil, errors.New("enum field not found or invalid in machineType")
	}
	return updateParams, nil
}
