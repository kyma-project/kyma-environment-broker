package command

import (
	"encoding/json"
	"fmt"
	broker "skr-tester/pkg/broker"
	kcp "skr-tester/pkg/kcp"
	"skr-tester/pkg/logger"

	"github.com/spf13/cobra"
)

type UpdateCommand struct {
	cobraCmd             *cobra.Command
	log                  logger.Logger
	instanceID           string
	planID               string
	updateMachineType    bool
	updateOIDC           bool
	updateAdministrators bool
	customMachineType    string
	customOIDC           string
	customAdministrators []string
}

func NewUpdateCommand() *cobra.Command {
	cmd := UpdateCommand{}
	cobraCmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"u"},
		Short:   "Update the instance",
		Long:    "Update the instance with a new machine type, OIDC configuration, or administrators.",
		Example: `	skr-tester update -i instanceID -p planID --updateMachineType             Update the instance with a new machine type.
	skr-tester update -i instanceID -p planID --updateOIDC                    Update the instance with a new OIDC configuration.
	skr-tester update -i instanceID -p planID --updateAdministrators          Update the instance with new administrators.
	skr-tester update -i instanceID -p planID --updateMachineType --machineType newMachineType                                             Update the instance with a custom machine type.
	skr-tester update -i instanceID -p planID --updateOIDC --customOIDC '{"clientID":"foo-bar","issuerURL":"https://new.custom.ias.com"}'  Update the instance with a custom OIDC configuration.
	skr-tester update -i instanceID -p planID --updateAdministrators --customAdministrators admin1@acme.com,admin2@acme.com                Update the instance with custom administrators.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd
	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "Instance ID of the specific instance.")
	cobraCmd.Flags().StringVarP(&cmd.planID, "planID", "p", "", "Plan ID of the specific instance.")
	cobraCmd.Flags().BoolVarP(&cmd.updateMachineType, "updateMachineType", "m", false, "Update machine type.")
	cobraCmd.Flags().BoolVarP(&cmd.updateOIDC, "updateOIDC", "o", false, "Update OIDC configuration.")
	cobraCmd.Flags().BoolVarP(&cmd.updateAdministrators, "updateAdministrators", "a", false, "Update administrators.")
	cobraCmd.Flags().StringVar(&cmd.customMachineType, "customMachineType", "", "Machine type to update to (optional).")
	cobraCmd.Flags().StringVar(&cmd.customOIDC, "customOIDC", "", "Custom OIDC configuration in JSON format (optional).")
	cobraCmd.Flags().StringSliceVar(&cmd.customAdministrators, "customAdministrators", nil, "Custom administrators (optional).")

	return cobraCmd
}

func (cmd *UpdateCommand) Run() error {
	cmd.log = logger.New()
	brokerClient := broker.NewBrokerClient(broker.NewBrokerConfig())
	kcpClient, err := kcp.NewKCPClient()
	if err != nil {
		return fmt.Errorf("failed to create KCP client: %v", err)
	}
	if cmd.updateMachineType {
		if cmd.customMachineType != "" {
			fmt.Printf("User provided machine type: %s\n", cmd.customMachineType)
			resp, _, err := brokerClient.UpdateInstance(cmd.instanceID, map[string]interface{}{"machineType": cmd.customMachineType})
			if err != nil {
				return fmt.Errorf("error updating instance: %v", err)
			}
			fmt.Printf("Update operationID: %s\n", resp["operation"].(string))
			return nil
		}

		catalog, _, err := brokerClient.GetCatalog()
		if err != nil {
			return fmt.Errorf("failed to get catalog: %v", err)
		}
		services, ok := catalog["services"].([]interface{})
		if !ok {
			return fmt.Errorf("services field not found or invalid in catalog")
		}
		for _, service := range services {
			serviceMap, ok := service.(map[string]interface{})
			if !ok {
				return fmt.Errorf("service is not a map[string]interface{}")
			}
			if serviceMap["id"] != broker.KymaServiceID {
				continue
			}

			currentMachineType, err := kcpClient.GetCurrentMachineType(cmd.instanceID)
			if err != nil {
				return fmt.Errorf("failed to get current machine type: %v", err)
			}
			fmt.Printf("Current machine type: %s\n", *currentMachineType)
			plans, ok := serviceMap["plans"].([]interface{})
			if !ok {
				return fmt.Errorf("plans field not found or invalid in serviceMap")
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
						resp, _, err := brokerClient.UpdateInstance(cmd.instanceID, map[string]interface{}{"machineType": newMachineType})
						if err != nil {
							return fmt.Errorf("error updating instance: %v", err)
						}
						fmt.Printf("Update operationID: %s\n", resp["operation"].(string))
						return nil
					}
				}

			}
		}
	} else if cmd.updateOIDC {
		if cmd.customOIDC != "" {
			var newOIDCConfig map[string]interface{}
			err := json.Unmarshal([]byte(cmd.customOIDC), &newOIDCConfig)
			if err != nil {
				return fmt.Errorf("failed to parse custom OIDC config: %v", err)
			}
			fmt.Printf("User provided custom OIDC config: %v\n", newOIDCConfig)
			resp, _, err := brokerClient.UpdateInstance(cmd.instanceID, map[string]interface{}{"oidc": newOIDCConfig})
			if err != nil {
				return fmt.Errorf("error updating instance: %v", err)
			}
			fmt.Printf("Update operationID: %s\n", resp["operation"].(string))
			return nil
		}

		currentOIDCConfig, err := kcpClient.GetCurrentOIDCConfig(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get current OIDC config: %v", err)
		}
		fmt.Printf("Current OIDC config: %v\n", currentOIDCConfig)
		newOIDCConfig := map[string]interface{}{
			"clientID":       "foo-bar",
			"groupsClaim":    "groups1",
			"issuerURL":      "https://new.custom.ias.com",
			"signingAlgs":    []string{"RS256"},
			"usernameClaim":  "email",
			"usernamePrefix": "acme-",
		}
		fmt.Printf("Determined OIDC configuration to update: %v\n", newOIDCConfig)
		resp, _, err := brokerClient.UpdateInstance(cmd.instanceID, map[string]interface{}{"oidc": newOIDCConfig})
		if err != nil {
			return fmt.Errorf("error updating instance: %v", err)
		}
		fmt.Printf("Update operationID: %s\n", resp["operation"].(string))
	} else if cmd.updateAdministrators {
		if len(cmd.customAdministrators) > 0 {
			fmt.Printf("User provided custom administrators: %v\n", cmd.customAdministrators)
			resp, _, err := brokerClient.UpdateInstance(cmd.instanceID, map[string]interface{}{"administrators": cmd.customAdministrators})
			if err != nil {
				return fmt.Errorf("error updating instance: %v", err)
			}
			fmt.Printf("Update operationID: %s\n", resp["operation"].(string))
			return nil
		}

		newAdministrators := []string{"admin1@acme.com", "admin2@acme.com"}
		fmt.Printf("Determined administrators to update: %v\n", newAdministrators)
		resp, _, err := brokerClient.UpdateInstance(cmd.instanceID, map[string]interface{}{"administrators": newAdministrators})
		if err != nil {
			return fmt.Errorf("error updating instance: %v", err)
		}
		fmt.Printf("Update operationID: %s\n", resp["operation"].(string))
	}
	return nil
}

func (cmd *UpdateCommand) Validate() error {
	if cmd.instanceID == "" || cmd.planID == "" {
		return fmt.Errorf("you must specify the planID and instanceID")
	}
	updateCount := 0
	if cmd.updateMachineType {
		updateCount++
	}
	if cmd.updateOIDC {
		updateCount++
	}
	if cmd.updateAdministrators {
		updateCount++
	}
	if updateCount != 1 {
		return fmt.Errorf("you must use exactly one of updateMachineType, updateOIDC, or updateAdministrators")
	}
	return nil
}

func extractSupportedMachineTypes(planMap map[string]interface{}) ([]interface{}, error) {
	schemas, ok := planMap["schemas"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schemas field not found or invalid in planMap")
	}
	serviceInstance, ok := schemas["service_instance"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("service_instance field not found or invalid in schemas")
	}
	update, ok := serviceInstance["update"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("update field not found or invalid in service_instance")
	}
	parameters, ok := update["parameters"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parameters field not found or invalid in update")
	}
	properties, ok := parameters["properties"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("properties field not found or invalid in parameters")
	}
	machineType, ok := properties["machineType"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("machineType field not found or invalid in properties")
	}
	supportedMachineTypes, ok := machineType["enum"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("enum field not found or invalid in machineType")
	}
	return supportedMachineTypes, nil
}
