package main

import (
	"fmt"
	"os"
	"os/signal"
	keb "skr-tester/internal/kyma-environment-broker"
	"syscall"
)

func main() {
	setupCloseHandler()

	kebClient := keb.NewKEBClient(keb.NewKEBConfig())
	/*dummyCreds := map[string]interface{}{
		"clientid":     "dummy_client_id",
		"clientsecret": "dummy_client_secret",
		"smURL":        "dummy_url",
		"url":          "dummy_token_url",
	}
	resp, err := kebClient.ProvisionSKR("mm-24122024", "mm-24122024", nil, dummyCreds, nil)
	if err != nil {
		fmt.Printf("Error provisioning SKR: %v\n", err)
	} else {
		fmt.Printf("Provisioning response: %v\n", resp)
	}
	*/
	deprovisionResp, err := kebClient.DeprovisionSKR("mm-24122024")
	if err != nil {
		fmt.Printf("Error deprovisioning SKR: %v\n", err)
	} else {
		fmt.Printf("Deprovisioning response: %v\n", deprovisionResp)
	}

}

func setupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-c
		fmt.Printf("\r- Signal '%v' received from Terminal. Exiting...\n ", sig)
		os.Exit(0)
	}()
}
