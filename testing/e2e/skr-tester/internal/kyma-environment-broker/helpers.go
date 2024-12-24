package keb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type KEB interface {
	ProvisionSKR(name, instanceID string, platformCreds, btpOperatorCreds, customParams interface{}) (map[string]interface{}, error)
	DeprovisionSKR(instanceID string) (map[string]interface{}, error)
	UpdateSKR(instanceID string, customParams, btpOperatorCreds interface{}, isMigration bool) (map[string]interface{}, error)
	GetOperation(instanceID, operationID string) (map[string]interface{}, error)
	GetCatalog() (map[string]interface{}, error)
	DownloadKubeconfig(instanceID string) (string, error)
}

type KCP interface {
	GetKubeconfig(shootName string) (string, error)
	GetRuntimeGardenerConfig(shootName string) (string, error)
	GetRuntimeStatusOperations(instanceID string) (string, error)
	GetRuntimeEvents(instanceID string) (string, error)
}

type Gardener interface {
	GetShoot(shootName string) (map[string]interface{}, error)
}

func ProvisionSKR(keb KEB, instanceID, name string, platformCreds, btpOperatorCreds, customParams interface{}, timeout time.Duration) error {
	resp, err := keb.ProvisionSKR(name, instanceID, platformCreds, btpOperatorCreds, customParams)
	if err != nil {
		return err
	}
	if _, ok := resp["operation"]; !ok {
		return fmt.Errorf("operation key not found in response")
	}

	operationID := resp["operation"].(string)
	fmt.Printf("Operation ID %s\n", operationID)

	//return EnsureOperationSucceeded(keb, kcp, instanceID, operationID, timeout)
	return nil
}

func GetShoot(kcp KCP, shootName string) (map[string]interface{}, error) {
	fmt.Printf("Fetching shoot: %s\n", shootName)

	kubeconfigPath, err := kcp.GetKubeconfig(shootName)
	if err != nil {
		return nil, err
	}

	runtimeGardenerConfig, err := kcp.GetRuntimeGardenerConfig(shootName)
	if err != nil {
		return nil, err
	}

	var objRuntimeGardenerConfig map[string]interface{}
	err = json.Unmarshal([]byte(runtimeGardenerConfig), &objRuntimeGardenerConfig)
	if err != nil {
		return nil, err
	}

	data := objRuntimeGardenerConfig["data"].([]interface{})[0].(map[string]interface{})
	status := data["status"].(map[string]interface{})

	if gardenerConfig, ok := status["gardenerConfig"].(map[string]interface{}); ok {
		if gardenerConfig["oidcConfig"] == nil || gardenerConfig["machineType"] == nil {
			return nil, fmt.Errorf("oidcConfig or machineType is empty")
		}
		return map[string]interface{}{
			"name":        shootName,
			"kubeconfig":  kubeconfigPath,
			"oidcConfig":  gardenerConfig["oidcConfig"],
			"machineType": gardenerConfig["machineType"],
		}, nil
	}

	runtimeConfig := data["runtimeConfig"].(map[string]interface{})
	spec := runtimeConfig["spec"].(map[string]interface{})
	shoot := spec["shoot"].(map[string]interface{})
	kubernetes := shoot["kubernetes"].(map[string]interface{})
	kubeAPIServer := kubernetes["kubeAPIServer"].(map[string]interface{})
	oidcConfig := kubeAPIServer["oidcConfig"]

	provider := shoot["provider"].(map[string]interface{})
	workers := provider["workers"].([]interface{})[0].(map[string]interface{})
	machine := workers["machine"].(map[string]interface{})
	machineType := machine["type"]

	if oidcConfig == nil || machineType == nil {
		return nil, fmt.Errorf("oidcConfig or machineType is empty")
	}

	return map[string]interface{}{
		"name":        shootName,
		"kubeconfig":  kubeconfigPath,
		"oidcConfig":  oidcConfig,
		"machineType": machineType,
	}, nil
}

func EnsureValidShootOIDCConfig(shoot, targetOIDCConfig map[string]interface{}) error {
	if shoot["oidcConfig"].(map[string]interface{})["clientID"] != targetOIDCConfig["clientID"] ||
		shoot["oidcConfig"].(map[string]interface{})["issuerURL"] != targetOIDCConfig["issuerURL"] ||
		shoot["oidcConfig"].(map[string]interface{})["groupsClaim"] != targetOIDCConfig["groupsClaim"] ||
		shoot["oidcConfig"].(map[string]interface{})["usernameClaim"] != targetOIDCConfig["usernameClaim"] ||
		shoot["oidcConfig"].(map[string]interface{})["usernamePrefix"] != targetOIDCConfig["usernamePrefix"] ||
		shoot["oidcConfig"].(map[string]interface{})["signingAlgs"] != targetOIDCConfig["signingAlgs"] {
		return fmt.Errorf("OIDC config does not match")
	}
	return nil
}

func DeprovisionSKR(keb KEB, kcp KCP, instanceID string, timeout time.Duration, ensureSuccess bool) (string, error) {
	resp, err := keb.DeprovisionSKR(instanceID)
	if err != nil {
		return "", err
	}
	if _, ok := resp["operation"]; !ok {
		return "", fmt.Errorf("operation key not found in response")
	}

	operationID := resp["operation"].(string)
	fmt.Printf("Deprovision SKR - operation ID %s\n", operationID)

	if ensureSuccess {
		err = EnsureOperationSucceeded(keb, kcp, instanceID, operationID, timeout)
		if err != nil {
			return "", err
		}
	}

	return operationID, nil
}

func UpdateSKR(keb KEB, kcp KCP, gardener Gardener, instanceID, shootName string, customParams, btpOperatorCreds interface{}, timeout time.Duration, isMigration bool) (map[string]interface{}, error) {
	resp, err := keb.UpdateSKR(instanceID, customParams, btpOperatorCreds, isMigration)
	if err != nil {
		return nil, err
	}
	if _, ok := resp["operation"]; !ok {
		return nil, fmt.Errorf("operation key not found in response")
	}

	operationID := resp["operation"].(string)
	fmt.Printf("Operation ID %s\n", operationID)

	err = EnsureOperationSucceeded(keb, kcp, instanceID, operationID, timeout)
	if err != nil {
		return nil, err
	}

	var shoot map[string]interface{}
	if os.Getenv("GARDENER_KUBECONFIG") != "" {
		shoot, err = gardener.GetShoot(shootName)
	} else {
		shoot, err = GetShoot(kcp, shootName)
	}
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"operationID": operationID,
		"shoot":       shoot,
	}, nil
}

func EnsureOperationSucceeded(keb KEB, kcp KCP, instanceID, operationID string, timeout time.Duration) error {
	var res map[string]interface{}
	err := wait(func() (bool, error) {
		res, err := keb.GetOperation(instanceID, operationID)
		if err != nil {
			return false, err
		}
		state := res["state"].(string)
		return state == "succeeded" || state == "failed", nil
	}, timeout, 30*time.Second)
	if err != nil {
		runtimeStatus, _ := kcp.GetRuntimeStatusOperations(instanceID)
		events, _ := kcp.GetRuntimeEvents(instanceID)
		return fmt.Errorf("%v\nError thrown by EnsureOperationSucceeded: Runtime status: %s\nEvents:\n%s", err, runtimeStatus, events)
	}

	if res["state"].(string) != "succeeded" {
		runtimeStatus, _ := kcp.GetRuntimeStatusOperations(instanceID)
		return fmt.Errorf("Error thrown by EnsureOperationSucceeded: operation didn't succeed in time: %v\nRuntime status: %s", res, runtimeStatus)
	}

	fmt.Printf("Operation %s finished with state %s\n", operationID, res["state"].(string))
	return nil
}

func GetCatalog(keb KEB) (map[string]interface{}, error) {
	return keb.GetCatalog()
}

func EnsureValidOIDCConfigInCustomerFacingKubeconfig(keb KEB, instanceID string, oidcConfig map[string]interface{}) error {
	kubeconfigContent, err := keb.DownloadKubeconfig(instanceID)
	if err != nil {
		return err
	}

	issuerMatchPattern := fmt.Sprintf("\\b%s\\b", oidcConfig["issuerURL"])
	clientIDMatchPattern := fmt.Sprintf("\\b%s\\b", oidcConfig["clientID"])
	if !regexp.MustCompile(issuerMatchPattern).MatchString(kubeconfigContent) || !regexp.MustCompile(clientIDMatchPattern).MatchString(kubeconfigContent) {
		return fmt.Errorf("OIDC config does not match in kubeconfig")
	}

	return nil
}

func SaveKubeconfig(kubeconfig string) error {
	directory := filepath.Join(os.Getenv("HOME"), ".kube")
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.MkdirAll(directory, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return ioutil.WriteFile(filepath.Join(directory, "config"), []byte(kubeconfig), 0644)
}

func wait(condition func() (bool, error), timeout, interval time.Duration) error {
	start := time.Now()
	for {
		done, err := condition()
		if done {
			return err
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout after %v", timeout)
		}
		time.Sleep(interval)
	}
}
