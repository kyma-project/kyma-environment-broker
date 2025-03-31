package command

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	broker "skr-tester/pkg/broker"
	kcp "skr-tester/pkg/kcp"
	"skr-tester/pkg/logger"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sWait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	SKRAccessInterval = 10 * time.Second
	SKRAccessTimeout  = 2 * time.Minute
)

type AssertCommand struct {
	cobraCmd                  *cobra.Command
	log                       logger.Logger
	instanceID                string
	machineType               string
	clusterOIDCConfig         string
	kubeconfigOIDCConfig      []string
	admins                    []string
	btpManagerSecretExists    bool
	editBtpManagerSecret      bool
	deleteBtpManagerSecret    bool
	suspensionInProgress      bool
	endpointsSecured          bool
	additionalWorkerNodePools string
}

func NewAsertCmd() *cobra.Command {
	cmd := AssertCommand{}
	cobraCmd := &cobra.Command{
		Use:     "assert",
		Aliases: []string{"a"},
		Short:   "Performs an assertion",
		Long:    "Performs an assertion",
		Example: `  skr-tester assert -i instanceID -m m6i.large                           Asserts the instance has the machine type m6i.large.
  skr-tester assert -i instanceID -o oidcConfig                          Asserts the instance has the OIDC config equal to oidcConfig.
  skr-tester assert -i instanceID -k issuerURL,clientID                  Asserts the kubeconfig contains the specified issuerURL and clientID.
  skr-tester assert -i instanceID -a admin1,admin2                       Asserts the specified admins are present in the cluster role bindings.
  skr-tester assert -i instanceID -b                                     Checks if the BTP manager secret exists in the instance.
  skr-tester assert -i instanceID -e                                     Edits the BTP manager secret in the instance and checks if the secret is reconciled.
  skr-tester assert -i instanceID -d                                     Deletes the BTP manager secret in the instance and checks if the secret is reconciled.
  skr-tester assert -i instanceID -s                                     Checks if the suspension operation is in progress for the instance.
  skr-tester assert -i instanceID -n                                     Checks if KEB endpoints require authentication.
  skr-tester assert -i instanceID -w additionalWorkerNodePools           Asserts the instance has the specified additional worker node pools.`,

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "Instance ID of the specific instance.")
	cobraCmd.Flags().StringVarP(&cmd.machineType, "machineType", "m", "", "Asserts the instance has the specified machine type.")
	cobraCmd.Flags().StringVarP(&cmd.clusterOIDCConfig, "clusterOIDCConfig", "o", "", "Asserts the instance has the specified cluster OIDC config.")
	cobraCmd.Flags().StringSliceVarP(&cmd.kubeconfigOIDCConfig, "kubeconfigOIDCConfig", "k", nil, "Asserts the kubeconfig contains the specified issuer URL and client ID in the format issuerURL,clientID.")
	cobraCmd.Flags().StringSliceVarP(&cmd.admins, "admins", "a", nil, "Admins of the specific instance.")
	cobraCmd.Flags().BoolVarP(&cmd.btpManagerSecretExists, "btpManagerSecretExists", "b", false, "Checks if the BTP manager secret exists in the instance.")
	cobraCmd.Flags().BoolVarP(&cmd.editBtpManagerSecret, "editBtpManagerSecret", "e", false, "Edits the BTP manager secret in the instance and checks if the secret is reconciled.")
	cobraCmd.Flags().BoolVarP(&cmd.deleteBtpManagerSecret, "deleteBtpManagerSecret", "d", false, "Deletes the BTP manager secret in the instance and checks if the secret is reconciled.")
	cobraCmd.Flags().BoolVarP(&cmd.suspensionInProgress, "suspensionInProgress", "s", false, "Checks if the suspension operation is in progress for the instance.")
	cobraCmd.Flags().BoolVarP(&cmd.endpointsSecured, "endpointsSecured", "n", false, "Tests the KEB endpoints without authorization.")
	cobraCmd.Flags().StringVarP(&cmd.additionalWorkerNodePools, "additionalWorkerNodePools", "w", "", "Additional worker node pools of the specific instance.")

	return cobraCmd
}

func (cmd *AssertCommand) Run() error {
	cmd.log = logger.New()
	ctrl.SetLogger(zap.New())
	brokerClient := broker.NewBrokerClient(broker.NewBrokerConfig())
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
	} else if cmd.clusterOIDCConfig != "" {
		currentOIDC, err := kcpClient.GetCurrentOIDCConfig(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get current OIDC: %v", err)
		}
		if cmd.clusterOIDCConfig != fmt.Sprintf("%v", currentOIDC) {
			return fmt.Errorf("OIDCs are not equal: expected %s, got %s", cmd.clusterOIDCConfig, fmt.Sprintf("%v", currentOIDC))
		} else {
			fmt.Println("OIDC assertion passed: expected and got", cmd.clusterOIDCConfig)
		}
	} else if cmd.kubeconfigOIDCConfig != nil {
		kubeconfig, err := brokerClient.DownloadKubeconfig(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to download kubeconfig: %v", err)
		}
		issuerMatchPattern := fmt.Sprintf("\\b%s\\b", cmd.kubeconfigOIDCConfig[0])
		clientIDMatchPattern := fmt.Sprintf("\\b%s\\b", cmd.kubeconfigOIDCConfig[1])

		if !regexp.MustCompile(issuerMatchPattern).MatchString(kubeconfig) {
			return fmt.Errorf("issuerURL %s not found in kubeconfig", cmd.kubeconfigOIDCConfig[0])
		}
		if !regexp.MustCompile(clientIDMatchPattern).MatchString(kubeconfig) {
			return fmt.Errorf("clientID %s not found in kubeconfig", cmd.kubeconfigOIDCConfig[1])
		}
		fmt.Println("Kubeconfig OIDC assertion passed")

	} else if cmd.admins != nil {
		kubeconfig, err := kcpClient.GetKubeconfig(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig: %v", err)
		}
		k8sCli, err := cmd.newK8sClient(kubeconfig)
		if err != nil {
			return fmt.Errorf("while creating k8s client: %w", err)
		}
		clusterRoleBindings := &rbacv1.ClusterRoleBindingList{}
		err = k8sCli.List(context.TODO(), clusterRoleBindings, &client.ListOptions{})
		if err != nil {
			return fmt.Errorf("while listing cluster role bindings: %w", err)
		}
		adminsMap := make(map[string]bool)
		for _, admin := range cmd.admins {
			adminsMap[admin] = false
		}
		fmt.Println("Looking for the following admins:", cmd.admins)
		for _, crb := range clusterRoleBindings.Items {
			if crb.RoleRef.Name == "cluster-admin" {
				for _, subject := range crb.Subjects {
					if adminsMap[subject.Name] == false {
						adminsMap[subject.Name] = true
					}
				}
			}
		}
		for admin, found := range adminsMap {
			if !found {
				return fmt.Errorf("admin %s not found in cluster role bindings", admin)
			}
		}
		fmt.Println("All specified admins are found in cluster role bindings")
	} else if cmd.btpManagerSecretExists {
		kubeconfig, err := kcpClient.GetKubeconfig(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig: %v", err)
		}
		err = cmd.checkBTPManagerSecret(kubeconfig)
		if err != nil {
			return err
		}
	} else if cmd.deleteBtpManagerSecret {
		kubeconfig, err := kcpClient.GetKubeconfig(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig: %v", err)
		}
		k8sCli, err := cmd.newK8sClient(kubeconfig)
		if err != nil {
			return fmt.Errorf("while creating k8s client: %w", err)
		}
		secret := &v1.Secret{}
		objKey := client.ObjectKey{Namespace: "kyma-system", Name: "sap-btp-manager"}
		if err := k8sCli.Get(context.Background(), objKey, secret); err != nil {
			return fmt.Errorf("failed to get secret: %w", err)
		}
		err = k8sCli.Delete(context.Background(), secret)
		if err != nil {
			return fmt.Errorf("while deleting secret from instace: %w", err)
		}
		fmt.Println("BTP manager secret deleted successfully")
		retriesBeforeTimeout := 10
		for i := 0; i < retriesBeforeTimeout; i++ {
			time.Sleep(6 * time.Second)
			secret := &v1.Secret{}
			objKey := client.ObjectKey{Namespace: "kyma-system", Name: "sap-btp-manager"}
			if err := k8sCli.Get(context.Background(), objKey, secret); err != nil {
				if k8serrors.IsNotFound(err) {
					fmt.Printf("Waiting for the secret to be reconciled... (retry %d/%d)\n", i+1, retriesBeforeTimeout)
					continue
				}
				return fmt.Errorf("failed to get secret: %w", err)
			} else {
				break
			}
		}
		err = cmd.checkBTPManagerSecret(kubeconfig)
		if err != nil {
			return err
		}
		fmt.Println("BTP manager secret delete test passed")
	} else if cmd.editBtpManagerSecret {
		kubeconfig, err := kcpClient.GetKubeconfig(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig: %v", err)
		}
		k8sCli, err := cmd.newK8sClient(kubeconfig)
		if err != nil {
			return fmt.Errorf("while creating k8s client: %w", err)
		}
		secret := &v1.Secret{}
		objKey := client.ObjectKey{Namespace: "kyma-system", Name: "sap-btp-manager"}
		if err := k8sCli.Get(context.Background(), objKey, secret); err != nil {
			return fmt.Errorf("failed to get secret: %w", err)
		}
		secret.Data["clientid"] = []byte("new_client_id")
		secret.Data["clientsecret"] = []byte("new_client_secret")
		secret.Data["sm_url"] = []byte("new_url")
		secret.Data["tokenurl"] = []byte("new_token_url")
		err = k8sCli.Update(context.Background(), secret)
		if err != nil {
			return fmt.Errorf("while updating secret from instace: %w", err)
		}
		fmt.Println("BTP manager secret updated successfully")
		retriesBeforeTimeout := 100
		reconciledSecret := &v1.Secret{}
		for i := 0; i < retriesBeforeTimeout; i++ {
			time.Sleep(6 * time.Second)
			if err := k8sCli.Get(context.Background(), objKey, reconciledSecret); err != nil {
				return fmt.Errorf("failed to get secret: %w", err)
			}
			if reconciledSecret.ObjectMeta.Name == "sap-btp-manager" && reconciledSecret.ObjectMeta.ResourceVersion != secret.ObjectMeta.ResourceVersion {
				break
			}
			fmt.Printf("Waiting for the secret to be reconciled... (retry %d/%d)\n", i+1, retriesBeforeTimeout)
		}
		err = cmd.checkBTPManagerSecret(kubeconfig)
		if err != nil {
			return err
		}
		fmt.Println("BTP manager secret update test passed")
	} else if cmd.suspensionInProgress {
		retriesBeforeTimeout := 11
		var operationID *string
		for i := 0; i < retriesBeforeTimeout; i++ {
			operationID, err = kcpClient.GetSuspensionOperationID(cmd.instanceID)
			if err != nil {
				return fmt.Errorf("failed to get suspension status: %v", err)
			}
			if *operationID != "" {
				break
			}
			fmt.Printf("Waiting for the suspension operation to start... (retry %d/%d)\n", i+1, retriesBeforeTimeout)
			time.Sleep(time.Minute)
		}
		if *operationID == "" {
			return fmt.Errorf("suspension operation not found")
		}
		resp, _, err := brokerClient.GetOperation(cmd.instanceID, *operationID)
		if err != nil {
			return err
		}
		state, ok := resp["state"].(string)
		if !ok {
			return fmt.Errorf("state field not found in suspension operation response")
		}
		if state != "in progress" {
			return fmt.Errorf("suspension operation status is not 'in progress': %s", state)
		}
		fmt.Println("Suspension operation is in progress")
		fmt.Printf("Suspension operationID: %s\n", *operationID)
	} else if cmd.endpointsSecured {
		brokerClient := broker.NewBrokerClient(broker.NewBrokerConfig())
		platformRegion := brokerClient.GetPlatformRegion()
		testData := []struct {
			payload  interface{}
			endpoint string
			method   string
		}{
			{payload: nil, endpoint: fmt.Sprintf("oauth/v2/service_instances/%s", cmd.instanceID), method: "GET"},
			{payload: nil, endpoint: "runtimes", method: "GET"},
			{payload: nil, endpoint: "info/runtimes", method: "GET"},
			{payload: nil, endpoint: "orchestrations", method: "GET"},
			{payload: nil, endpoint: fmt.Sprintf("oauth/%sv2/service_instances/%s", platformRegion, cmd.instanceID), method: "PUT"},
			{payload: nil, endpoint: "upgrade/cluster", method: "POST"},
			{payload: nil, endpoint: "upgrade/kyma", method: "POST"},
			{payload: nil, endpoint: fmt.Sprintf("oauth/v2/service_instances/%s", cmd.instanceID), method: "PATCH"},
			{payload: nil, endpoint: fmt.Sprintf("oauth/v2/service_instances/%s", cmd.instanceID), method: "DELETE"},
		}

		for _, test := range testData {
			err := brokerClient.CallBrokerWithoutToken(test.payload, test.endpoint, test.method)
			if err != nil {
				return fmt.Errorf("error while calling KEB endpoint %q without authorization: %v", test.endpoint, err)
			}
		}
		fmt.Println("KEB endpoints test passed")
	} else if cmd.additionalWorkerNodePools != "" {
		currentAdditionalWorkerNodePools, err := kcpClient.GetAdditionalWorkerNodePools(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get current additional worker node pools: %v", err)
		}
		currentAdditionalWorkerNodePoolsSet := make(map[string]interface{}, len(currentAdditionalWorkerNodePools))
		for _, pool := range currentAdditionalWorkerNodePools {
			currentAdditionalWorkerNodePoolsSet[pool["name"].(string)] = pool
		}
		planName, err := kcpClient.GetPlanName(cmd.instanceID)
		if err != nil {
			return fmt.Errorf("failed to get plan name: %v", err)
		}
		var additionalWorkerNodePools []map[string]interface{}
		err = json.Unmarshal([]byte(cmd.additionalWorkerNodePools), &additionalWorkerNodePools)
		if err != nil {
			return fmt.Errorf("error unmarshaling additional worker node pools: %v", err)
		}
		fmt.Println("Looking for the following additional worker node pools:", additionalWorkerNodePools)
		if err = cmd.assertAdditionalWorkerNodePools(additionalWorkerNodePools, currentAdditionalWorkerNodePoolsSet, planName); err != nil {
			return err
		}
		fmt.Println("All specified additional worker node pools are found in the instance")
	}

	return nil
}

func (cmd *AssertCommand) Validate() error {
	if cmd.instanceID == "" {
		return fmt.Errorf("instanceID must be specified")
	}
	count := 0
	if cmd.machineType != "" {
		count++
	}
	if cmd.clusterOIDCConfig != "" {
		count++
	}
	if cmd.kubeconfigOIDCConfig != nil {
		count++
	}
	if cmd.admins != nil {
		count++
	}
	if cmd.btpManagerSecretExists {
		count++
	}
	if cmd.editBtpManagerSecret {
		count++
	}
	if cmd.deleteBtpManagerSecret {
		count++
	}
	if cmd.suspensionInProgress {
		count++
	}
	if cmd.endpointsSecured {
		count++
	}
	if cmd.additionalWorkerNodePools != "" {
		count++
	}
	if count != 1 {
		return fmt.Errorf("you must use exactly one of machineType, clusterOIDCConfig, kubeconfigOIDCConfig, admins, btpManagerSecretExists, editBtpManagerSecret, deleteBtpManagerSecret, suspensionInProgress, endpointsSecured or additionalWorkerNodePools")
	}
	return nil
}

func (cmd *AssertCommand) checkBTPManagerSecret(kubeconfig []byte) error {
	k8sCli, err := cmd.newK8sClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("while creating k8s client: %w", err)
	}
	secret := &v1.Secret{}
	objKey := client.ObjectKey{Namespace: "kyma-system", Name: "sap-btp-manager"}
	if err := k8sCli.Get(context.Background(), objKey, secret); err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}
	if secret.Labels["app.kubernetes.io/managed-by"] != "kcp-kyma-environment-broker" {
		return fmt.Errorf("secret label 'app.kubernetes.io/managed-by' is not 'kcp-kyma-environment-broker'")
	}
	fmt.Println("BTP manager secret exists")

	requiredKeys := []string{"clientid", "clientsecret", "sm_url", "tokenurl", "cluster_id"}
	for _, key := range requiredKeys {
		if _, exists := secret.Data[key]; !exists {
			return fmt.Errorf("secret data key %s not found", key)
		}
	}
	fmt.Println("Required keys exist in BTP manager secret")

	expectedCreds := map[string]string{
		"clientid":     "dummy_client_id",
		"clientsecret": "dummy_client_secret",
		"sm_url":       "dummy_url",
		"tokenurl":     "dummy_token_url",
	}
	for key, expectedValue := range expectedCreds {
		if actualValue, exists := secret.Data[key]; !exists || string(actualValue) != expectedValue {
			return fmt.Errorf("secret data key %s does not have the expected value: expected %s, got %s", key, expectedValue, string(actualValue))
		}
	}
	fmt.Println("Required keys have the expected values in BTP manager secret")
	return nil
}

func (cmd *AssertCommand) assertAdditionalWorkerNodePools(additionalWorkerNodePools []map[string]interface{}, currentAdditionalWorkerNodePoolsSet map[string]interface{}, planName string) error {
	if len(additionalWorkerNodePools) != len(currentAdditionalWorkerNodePoolsSet) {
		return fmt.Errorf("expected %d additional worker node pools, but found %d", len(additionalWorkerNodePools), len(currentAdditionalWorkerNodePoolsSet))
	}
	for _, additionalWorkerNodePool := range additionalWorkerNodePools {
		currentAdditionalWorkerNodePool, exists := currentAdditionalWorkerNodePoolsSet[additionalWorkerNodePool["name"].(string)]
		if !exists {
			return fmt.Errorf("additional worker node pool %s not found", additionalWorkerNodePool["name"].(string))
		}
		if additionalWorkerNodePool["machineType"] != currentAdditionalWorkerNodePool.(map[string]interface{})["machine"].(map[string]interface{})["type"] {
			return fmt.Errorf(
				"machineType expected to be %d, but found %d",
				additionalWorkerNodePool["machineType"],
				currentAdditionalWorkerNodePool.(map[string]interface{})["machine"].(map[string]interface{})["type"],
			)
		}
		if additionalWorkerNodePool["haZones"].(bool) && planName != "azure_lite" {
			if len(currentAdditionalWorkerNodePool.(map[string]interface{})["zones"].([]interface{})) != 3 {
				return fmt.Errorf("expected 3 zones, but found %d", len(currentAdditionalWorkerNodePool.(map[string]interface{})["zones"].([]interface{})))
			}
		} else {
			if len(currentAdditionalWorkerNodePool.(map[string]interface{})["zones"].([]interface{})) != 1 {
				return fmt.Errorf("expected 1 zones, but found %d", len(currentAdditionalWorkerNodePool.(map[string]interface{})["zones"].([]interface{})))
			}
		}
		if additionalWorkerNodePool["autoScalerMin"] != currentAdditionalWorkerNodePool.(map[string]interface{})["minimum"] {
			return fmt.Errorf(
				"autoScalerMin expected to be %d, but found %d",
				additionalWorkerNodePool["autoScalerMin"],
				currentAdditionalWorkerNodePool.(map[string]interface{})["minimum"],
			)
		}
		if additionalWorkerNodePool["autoScalerMax"] != currentAdditionalWorkerNodePool.(map[string]interface{})["maximum"] {
			return fmt.Errorf(
				"autoScalerMax expected to be %d, but found %d",
				additionalWorkerNodePool["autoScalerMax"],
				currentAdditionalWorkerNodePool.(map[string]interface{})["maximum"],
			)
		}
	}
	return nil
}

func (cmd *AssertCommand) newK8sClient(kubeconfig []byte) (client.Client, error) {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("while creating REST config from kubeconfig: %w", err)
	}
	k8sCli, err := client.New(restCfg, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), SKRAccessTimeout)
	defer cancel()

	return k8sCli, k8sWait.PollUntilContextTimeout(ctx, SKRAccessInterval, SKRAccessTimeout, true, func(ctx context.Context) (bool, error) {
		podList := &v1.PodList{}
		err := k8sCli.List(ctx, podList, &client.ListOptions{Namespace: "kyma-system"})
		if err != nil {
			fmt.Printf("Technical user access to the SKR is not yet granted. Retrying in %s: %v\n", SKRAccessInterval, err)
			return false, nil
		}
		return true, nil
	})
}
