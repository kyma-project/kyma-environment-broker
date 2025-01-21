package command

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	broker "skr-tester/pkg/broker"
	"skr-tester/pkg/logger"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BindingCommand struct {
	cobraCmd                 *cobra.Command
	log                      logger.Logger
	instanceID               string
	create                   bool
	expirationSeconds        int
	getByID                  string
	checkKubeconfigValidity  bool
	deleteByID               string
	deleteNonExistingByID    string
	getNonExistingByID       string
	deleteAndCheckKubeconfig bool
	checkExpirationBelowMin  bool
	checkExpirationAboveMax  bool
	createTwoTimesTheSame    bool
	createCheckConflict      bool
	createAboveLimit         bool
}

func NewBindingCmd() *cobra.Command {
	cmd := BindingCommand{}
	cobraCmd := &cobra.Command{
		Use:     "binding",
		Aliases: []string{"b"},
		Short:   "Provides operations for bindings",
		Long:    "Provides operations for bindings",
		Example: "skr-tester binding -i instanceID --create                           Creates a binding.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "InstanceID of the specific instance.")
	cobraCmd.Flags().BoolVar(&cmd.create, "create", false, "Create a binding.")
	cobraCmd.Flags().IntVar(&cmd.expirationSeconds, "expirationSeconds", 600, "Expiration time in seconds for the binding. Leave empty for default value.")
	cobraCmd.Flags().StringVar(&cmd.getByID, "getByID", "", "Get a binding by ID.")
	cobraCmd.Flags().BoolVar(&cmd.checkKubeconfigValidity, "checkKubeconfigValidity", false, "Check the validity of the kubeconfig created by a binding.")
	cobraCmd.Flags().StringVar(&cmd.deleteByID, "deleteByID", "", "Delete a binding by ID.")
	cobraCmd.Flags().StringVar(&cmd.deleteNonExistingByID, "deleteNonExistingByID", "", "Delete a non-existing binding.")
	cobraCmd.Flags().StringVar(&cmd.getNonExistingByID, "getNonExistingByID", "", "Get a non-existing binding.")
	cobraCmd.Flags().BoolVar(&cmd.deleteAndCheckKubeconfig, "deleteAndCheckKubeconfig", false, "Delete a binding and check if the kubeconfig is still valid.")
	cobraCmd.Flags().BoolVar(&cmd.checkExpirationBelowMin, "checkExpirationBelowMin", false, "Check if the expiration time below the minimum value is correctly handled.")
	cobraCmd.Flags().BoolVar(&cmd.checkExpirationAboveMax, "checkExpirationAboveMax", false, "Check if the expiration time above the maximum value is correctly handled.")
	cobraCmd.Flags().BoolVar(&cmd.createTwoTimesTheSame, "createTwoTimesTheSame", false, "Create a binding two times with the same ID.")
	cobraCmd.Flags().BoolVar(&cmd.createCheckConflict, "createCheckConflict", false, "Create a binding two times with the same ID but different expiration time.")
	cobraCmd.Flags().BoolVar(&cmd.createAboveLimit, "createAboveLimit", false, "Create more bindings than the maximum allowed limit.")

	return cobraCmd
}

func (cmd *BindingCommand) Run() error {
	cmd.log = logger.New()
	brokerClient := broker.NewBrokerClient(broker.NewBrokerConfig())

	if cmd.create {
		return cmd.createBinding(brokerClient)
	} else if cmd.getByID != "" {
		return cmd.getBinding(brokerClient)
	} else if cmd.checkKubeconfigValidity {
		return cmd.checkKubeconfig(brokerClient)
	} else if cmd.deleteByID != "" {
		return cmd.deleteBinding(brokerClient)
	} else if cmd.deleteNonExistingByID != "" {
		return cmd.deleteNonExistingBinding(brokerClient)
	} else if cmd.getNonExistingByID != "" {
		return cmd.getNonExistingBinding(brokerClient)
	} else if cmd.deleteAndCheckKubeconfig {
		return cmd.deleteAndCheckKubeconfigValidity(brokerClient)
	} else if cmd.checkExpirationBelowMin {
		return cmd.checkExpirationBelowMinValue(brokerClient)
	} else if cmd.checkExpirationAboveMax {
		return cmd.checkExpirationAboveMaxValue(brokerClient)
	} else if cmd.createTwoTimesTheSame {
		return cmd.createBindingTwice(brokerClient)
	} else if cmd.createCheckConflict {
		return cmd.createBindingCheckConflict(brokerClient)
	} else if cmd.createAboveLimit {
		return cmd.createBindingsAboveLimit(brokerClient)
	}

	return nil
}

func (cmd *BindingCommand) createBinding(brokerClient *broker.BrokerClient) error {
	bindingID := uuid.New().String()
	resp, statusCode, err := brokerClient.CreateBinding(cmd.instanceID, bindingID, cmd.expirationSeconds)
	if err != nil {
		return fmt.Errorf("error creating binding: %v, response: %v", err, resp)
	}
	if *statusCode != http.StatusCreated {
		return fmt.Errorf("error creating binding: received status code %d, expected %d, response: %v", *statusCode, http.StatusCreated, resp)
	}

	fmt.Printf("Created binding with ID: %s\n", bindingID)
	return nil
}

func (cmd *BindingCommand) getBinding(brokerClient *broker.BrokerClient) error {
	resp, statusCode, err := brokerClient.GetBinding(cmd.instanceID, cmd.getByID)
	if err != nil {
		return fmt.Errorf("error getting binding: %v, response: %v", err, resp)
	}
	if *statusCode != http.StatusOK {
		return fmt.Errorf("error getting binding: received status code %d, expected %d", *statusCode, http.StatusOK)
	}

	fmt.Println("Binding retrieved successfully.")
	return nil
}

func (cmd *BindingCommand) checkKubeconfig(brokerClient *broker.BrokerClient) error {
	bindingID := uuid.New().String()
	resp, _, err := brokerClient.CreateBinding(cmd.instanceID, bindingID, cmd.expirationSeconds)
	if err != nil {
		return fmt.Errorf("error creating binding: %v, response: %v", err, resp)
	}
	kubeconfig, ok := resp["credentials"].(map[string]interface{})["kubeconfig"].(string)
	if !ok {
		return errors.New("failed to parse kubeconfig from binding credentials")
	}
	fmt.Println("Testing kubeconfig returned in response from create binding.")
	if err := cmd.validateKubeconfig(kubeconfig); err != nil {
		return err
	}

	binding, _, err := brokerClient.GetBinding(cmd.instanceID, bindingID)
	if err != nil {
		return fmt.Errorf("error getting binding: %v", err)
	}
	kubeconfig, ok = binding["credentials"].(map[string]interface{})["kubeconfig"].(string)
	if !ok {
		return errors.New("failed to parse kubeconfig from binding credentials")
	}
	fmt.Println("Testing kubeconfig returned in response from get binding.")
	return cmd.validateKubeconfig(kubeconfig)
}

func (cmd *BindingCommand) validateKubeconfig(kubeconfig string) error {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return fmt.Errorf("while creating rest config from kubeconfig: %w", err)
	}
	k8sCli, err := client.New(restCfg, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return fmt.Errorf("while creating k8s client: %w", err)
	}
	secret := &v1.Secret{}
	objKey := client.ObjectKey{Namespace: "kyma-system", Name: "sap-btp-manager"}
	if err := k8sCli.Get(context.Background(), objKey, secret); err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}
	fmt.Println("Kubeconfig is valid.")
	return nil
}

func (cmd *BindingCommand) deleteBinding(brokerClient *broker.BrokerClient) error {
	resp, statusCode, err := brokerClient.DeleteBinding(cmd.instanceID, cmd.deleteByID)
	if err != nil {
		return fmt.Errorf("error deleting binding: %v, response: %v", err, resp)
	}
	if *statusCode != http.StatusOK {
		return fmt.Errorf("error deleting binding: received status code %d, expected %d", *statusCode, http.StatusOK)
	}

	fmt.Printf("Binding with ID %s deleted successfully.\n", cmd.deleteByID)
	return nil
}

func (cmd *BindingCommand) deleteNonExistingBinding(brokerClient *broker.BrokerClient) error {
	resp, statusCode, err := brokerClient.DeleteBinding(cmd.instanceID, cmd.deleteNonExistingByID)
	if err != nil {
		if *statusCode != http.StatusGone {
			return fmt.Errorf("error deleting binding: received status code %d, expected %d, error: %v", *statusCode, http.StatusGone, err)
		}
	} else {
		return fmt.Errorf("expected error for deleting non-existing binding, but got nil. Status code: %d, response: %v", *statusCode, resp)
	}
	fmt.Printf("Attempted to delete a non-existing binding and received the expected status code: %d.\n", *statusCode)
	return nil
}

func (cmd *BindingCommand) getNonExistingBinding(brokerClient *broker.BrokerClient) error {
	resp, statusCode, err := brokerClient.GetBinding(cmd.instanceID, cmd.getNonExistingByID)
	if err != nil {
		if *statusCode != http.StatusNotFound {
			return fmt.Errorf("error getting binding: received status code %d, expected %d, error: %v", *statusCode, http.StatusNotFound, err)
		}
	} else {
		return fmt.Errorf("expected error for getting non-existing binding, but got nil. Status code: %d", *statusCode)
	}
	fmt.Printf("Attempted to get a non-existing binding and received the expected status code: %d, response: %v.\n", *statusCode, resp)
	return nil
}

func (cmd *BindingCommand) deleteAndCheckKubeconfigValidity(brokerClient *broker.BrokerClient) error {
	bindingID := uuid.New().String()
	fmt.Printf("Creating binding with ID: %s for test.\n", bindingID)
	resp, _, err := brokerClient.CreateBinding(cmd.instanceID, bindingID, cmd.expirationSeconds)
	if err != nil {
		return fmt.Errorf("error creating binding: %v", err)
	}
	kubeconfig, ok := resp["credentials"].(map[string]interface{})["kubeconfig"].(string)
	if !ok {
		return errors.New("failed to parse kubeconfig from binding credentials")
	}
	if err := cmd.validateKubeconfig(kubeconfig); err != nil {
		return err
	}
	_, statusCode, err := brokerClient.DeleteBinding(cmd.instanceID, bindingID)
	if err != nil {
		return fmt.Errorf("error deleting binding: %v", err)
	}
	if *statusCode != http.StatusOK {
		return fmt.Errorf("error deleting binding: received status code %d, expected %d", *statusCode, http.StatusOK)
	}

	fmt.Printf("Binding with ID %s deleted successfully.\n", bindingID)
	if err := cmd.validateKubeconfig(kubeconfig); err == nil {
		return errors.New("expected kubeconfig to be invalid after binding deletion, but it is still valid")
	} else if !strings.Contains(err.Error(), "failed to get secret: secrets") {
		return fmt.Errorf("unexpected error: %v", err)
	}
	fmt.Println("Test passed: Kubeconfig is invalid after binding deletion.")
	return nil
}

func (cmd *BindingCommand) checkExpirationBelowMinValue(brokerClient *broker.BrokerClient) error {
	bindingID := uuid.New().String()
	resp, statusCode, err := brokerClient.CreateBinding(cmd.instanceID, bindingID, 1)
	if err != nil {
		if *statusCode != http.StatusBadRequest {
			return fmt.Errorf("error creating binding: received status code %d, expected %d", *statusCode, http.StatusBadRequest)
		}
		if description, ok := resp["description"].(string); ok && strings.Contains(description, "expiration_seconds cannot be less than") {
			fmt.Printf("Attempted to create a binding with expiration time below the minimum value and received the expected error message: %s\n", description)
			return nil
		}
		return fmt.Errorf("error creating binding: %v", err)
	}
	fmt.Println("Expected an error for creating a binding with expiration time below the minimum value, but did not receive one.")
	return nil
}

func (cmd *BindingCommand) checkExpirationAboveMaxValue(brokerClient *broker.BrokerClient) error {
	bindingID := uuid.New().String()
	resp, statusCode, err := brokerClient.CreateBinding(cmd.instanceID, bindingID, 1000000000)
	if err != nil {
		if *statusCode != http.StatusBadRequest {
			return fmt.Errorf("error creating binding: received status code %d, expected %d", *statusCode, http.StatusBadRequest)
		}
		if description, ok := resp["description"].(string); ok && strings.Contains(description, "expiration_seconds cannot be greater than") {
			fmt.Printf("Attempted to create a binding with expiration time above the maximum value and received the expected error message: %s\n", description)
			return nil
		}
		return fmt.Errorf("error creating binding: %v", err)
	}
	fmt.Println("Expected an error for creating a binding with expiration time above the maximum value, but did not receive one.")
	return nil
}

func (cmd *BindingCommand) createBindingTwice(brokerClient *broker.BrokerClient) error {
	bindingID := uuid.New().String()
	_, statusCode, err := brokerClient.CreateBinding(cmd.instanceID, bindingID, cmd.expirationSeconds)
	if err != nil {
		return fmt.Errorf("error creating binding: %v", err)
	}
	if *statusCode != http.StatusCreated {
		return fmt.Errorf("error creating binding: received status code %d, expected %d", *statusCode, http.StatusCreated)
	}
	fmt.Printf("Created first binding with ID: %s, status code: %d\n", bindingID, *statusCode)
	_, statusCode, err = brokerClient.CreateBinding(cmd.instanceID, bindingID, cmd.expirationSeconds)
	if err != nil {
		return fmt.Errorf("error creating binding: %v", err)
	}
	if *statusCode != http.StatusOK {
		return fmt.Errorf("error creating binding: received status code %d, expected %d", *statusCode, http.StatusOK)
	}
	fmt.Printf("Created second binding with ID: %s, status code: %d\n", bindingID, *statusCode)
	fmt.Println("Attempted to create a binding with the same ID twice and received the expected status code.")
	return nil
}

func (cmd *BindingCommand) createBindingCheckConflict(brokerClient *broker.BrokerClient) error {
	bindingID := uuid.New().String()
	_, statusCode, err := brokerClient.CreateBinding(cmd.instanceID, bindingID, 800)
	if err != nil {
		return fmt.Errorf("error creating binding: %v", err)
	}
	if *statusCode != http.StatusCreated {
		return fmt.Errorf("error creating binding: received status code %d, expected %d", *statusCode, http.StatusCreated)
	}
	fmt.Printf("Created first binding with ID: %s, status code: %d\n", bindingID, *statusCode)
	_, statusCode, err = brokerClient.CreateBinding(cmd.instanceID, bindingID, 801)
	if err != nil {
		if *statusCode != http.StatusConflict {
			return fmt.Errorf("error creating binding: received status code %d, expected %d", *statusCode, http.StatusConflict)
		}
	}
	fmt.Printf("Tried to create second binding with ID: %s, status code: %d\n", bindingID, *statusCode)
	fmt.Println("Attempted to create a binding with the same ID but different expiration time and received the expected conflict status code.")
	return nil
}

func (cmd *BindingCommand) createBindingsAboveLimit(brokerClient *broker.BrokerClient) error {
	for i := 0; i < 13; i++ {
		bindingID := uuid.New().String()
		resp, statusCode, err := brokerClient.CreateBinding(cmd.instanceID, bindingID, cmd.expirationSeconds)
		if err != nil {
			if *statusCode == http.StatusBadRequest && strings.Contains(resp["description"].(string), "maximum number of non expired bindings reached") {
				fmt.Printf("Received expected error message for exceeding maximum number of non expired bindings. Status code: %d, description: %s\n", *statusCode, resp["description"].(string))
				return nil
			}
			return fmt.Errorf("error creating binding %d: %v, response: %v", i, err, resp)
		}
		fmt.Printf("Binding with ID %s created successfully.\n", bindingID)
	}
	return fmt.Errorf("created more bindings than the maximum allowed limit")
}

func (cmd *BindingCommand) Validate() error {
	if cmd.instanceID == "" {
		return errors.New("instanceID must be specified")
	}
	count := 0
	if cmd.create {
		count++
	}
	if cmd.getByID != "" {
		count++
	}
	if cmd.checkKubeconfigValidity {
		count++
	}
	if cmd.deleteByID != "" {
		count++
	}
	if cmd.deleteNonExistingByID != "" {
		count++
	}
	if cmd.getNonExistingByID != "" {
		count++
	}
	if cmd.deleteAndCheckKubeconfig {
		count++
	}
	if cmd.checkExpirationBelowMin {
		count++
	}
	if cmd.checkExpirationAboveMax {
		count++
	}
	if cmd.createTwoTimesTheSame {
		count++
	}
	if cmd.createCheckConflict {
		count++
	}
	if cmd.createAboveLimit {
		count++
	}
	if count != 1 {
		return errors.New("you must use exactly one of create, getByID, checkKubeconfigValidity, deleteByID, deleteNonExistingByID, getNonExistingByID, deleteAndCheckKubeconfig, checkExpirationBelowMin, checkExpirationAboveMax, createTwoTimesTheSame, createCheckConflict, or createAboveLimit")
	}
	return nil
}
