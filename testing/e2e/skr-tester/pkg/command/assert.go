package command

import (
	"context"
	"errors"
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
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type AssertCommand struct {
	cobraCmd               *cobra.Command
	log                    logger.Logger
	instanceID             string
	machineType            string
	clusterOIDCConfig      string
	kubeconfigOIDCConfig   []string
	admins                 []string
	btpManagerSecretExists bool
	editBtpManagerSecret   bool
	deleteBtpManagerSecret bool
	suspensionInProgress   bool
}

func NewAsertCmd() *cobra.Command {
	cmd := AssertCommand{}
	cobraCmd := &cobra.Command{
		Use:     "assert",
		Aliases: []string{"a"},
		Short:   "Does an assertion",
		Long:    "Does an assertion",
		Example: "skr-tester assert -i instanceID -m m6i.large                           Asserts the instance has the machine type m6i.large.\n" +
			"skr-tester assert -i instanceID -o oidcConfig                          Asserts the instance has the OIDC config equal to oidcConfig.\n" +
			"skr-tester assert -i instanceID -k issuerURL,clientID                  Asserts the kubeconfig contains the specified issuerURL and clientID.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "InstanceID of the specific instance.")
	cobraCmd.Flags().StringVarP(&cmd.machineType, "machineType", "m", "", "MachineType of the specific instance.")
	cobraCmd.Flags().StringVarP(&cmd.clusterOIDCConfig, "clusterOIDCConfig", "o", "", "clusterOIDCConfig of the specific instance.")
	cobraCmd.Flags().StringSliceVarP(&cmd.kubeconfigOIDCConfig, "kubeconfigOIDCConfig", "k", nil, "kubeconfigOIDCConfig of the specific instance. Pass the issuerURL and clientID in the format issuerURL,clientID")
	cobraCmd.Flags().StringSliceVarP(&cmd.admins, "admins", "a", nil, "Admins of the specific instance.")
	cobraCmd.Flags().BoolVarP(&cmd.btpManagerSecretExists, "btpManagerSecretExists", "b", false, "Checks if the BTP manager secret exists in the instance.")
	cobraCmd.Flags().BoolVarP(&cmd.editBtpManagerSecret, "editBtpManagerSecret", "e", false, "Edits the BTP manager secret in the instance and checks if the secret is reconciled.")
	cobraCmd.Flags().BoolVarP(&cmd.deleteBtpManagerSecret, "deleteBtpManagerSecret", "d", false, "Deletes the BTP manager secret in the instance and checks if the secret is reconciled.")
	cobraCmd.Flags().BoolVarP(&cmd.suspensionInProgress, "suspensionInProgress", "s", false, "Checks if the suspension operation is in progress for the instance.")

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
		restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
		if err != nil {
			return fmt.Errorf("while creating rest config from kubeconfig: %w", err)
		}
		k8sCli, err := client.New(restCfg, client.Options{
			Scheme: scheme.Scheme,
		})
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
		restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
		if err != nil {
			return fmt.Errorf("while creating REST config from kubeconfig: %w", err)
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
		restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
		if err != nil {
			return fmt.Errorf("while creating REST config from kubeconfig: %w", err)
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
	}
	return nil
}

func (cmd *AssertCommand) Validate() error {
	if cmd.instanceID == "" {
		return errors.New("instanceID must be specified")
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
	if count != 1 {
		return errors.New("you must use exactly one of machineType, clusterOIDCConfig, kubeconfigOIDCConfig, admins, btpManagerSecretExists, editBtpManagerSecret, deleteBtpManagerSecret, or suspensionInProgress")
	}
	return nil
}

func (cmd *AssertCommand) checkBTPManagerSecret(kubeconfig []byte) error {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
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
