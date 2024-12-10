package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	EDP edp.Config
}

var edpClient *edp.Client
var configuration Config
var subaccountId string

func main() {

	subaccountId = os.Args[2]

	err := envconfig.InitWithPrefix(&configuration, "APP")
	if err != nil {
		panic(err)
	}
	edpClient = edp.NewClient(configuration.EDP)

	switch os.Args[1] {
	case "get":
		if len(os.Args) != 3 {
			help()
			return
		}
		GetCommand()
	case "register":
		if len(os.Args) != 5 {
			help()
			return
		}
		RegisterCommand()
	case "deregister":
		DeregisterCommand()
	}
}

func help() {
	fmt.Println("Usage: ./edp <command> <subaccountid> <platformregion> free|tdd|standard")
	fmt.Println("Available commands: get, register, deregister")
	fmt.Println("Examples:")
	fmt.Println("./edp get 123dc-345gh")
	fmt.Println("./edp register 123dc-345gh cf-eu21 standard")
}

func DeregisterCommand() {
	fmt.Println("Delete DataTenant metadata")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	subAccountID := strings.ToLower(subaccountId)
	for _, key := range []string{
		edp.MaasConsumerEnvironmentKey,
		edp.MaasConsumerRegionKey,
		edp.MaasConsumerSubAccountKey,
		edp.MaasConsumerServicePlan,
	} {
		err := edpClient.DeleteMetadataTenant(subAccountID, configuration.EDP.Environment, key, log)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Delete DataTenant")
	err := edpClient.DeleteDataTenant(subAccountID, configuration.EDP.Environment, log)
	if err != nil {
		panic(err)
	}

}

func RegisterCommand() {
	platformRegion := os.Args[3]
	plan := os.Args[4]

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	fmt.Println("Creating data tenant")
	err := edpClient.CreateDataTenant(edp.DataTenantPayload{
		Name:        subaccountId,
		Environment: configuration.EDP.Environment,
		Secret:      generateSecret(subaccountId, configuration.EDP.Environment),
	}, log)
	if err != nil {
		fmt.Println("Unable to create data tenant")
		panic(err)
	}

	fmt.Println("Creating metadata")
	for key, value := range map[string]string{
		edp.MaasConsumerEnvironmentKey: selectEnvironmentKey(platformRegion),
		edp.MaasConsumerRegionKey:      platformRegion,
		edp.MaasConsumerSubAccountKey:  subaccountId,
		edp.MaasConsumerServicePlan:    plan,
	} {
		payload := edp.MetadataTenantPayload{
			Key:   key,
			Value: value,
		}
		fmt.Printf("Sending metadata %s: %s\n", payload.Key, payload.Value)
		err = edpClient.CreateMetadataTenant(subaccountId, configuration.EDP.Environment, payload, log)
		if err != nil {
			if edp.IsConflictError(err) {
				fmt.Println("Metadata already exists.")
				return
			}
			fmt.Printf("cannot create DataTenant metadata %s\n", key)
			return
		}
	}
}

func generateSecret(name, env string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%s", name, env)))
}

func GetCommand() {
	items, err := edpClient.GetMetadataTenant(subaccountId, configuration.EDP.Environment)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	for _, item := range items {
		fmt.Printf("%s %s %s\n", item.Key, item.Value, item.DataTenant.Name)
	}
	if len(items) == 0 {
		fmt.Println("Not found")
	}
}

func selectServicePlan(planID string) string {
	switch planID {
	case broker.FreemiumPlanID:
		return "free"
	case broker.AzureLitePlanID:
		return "tdd"
	default:
		return "standard"
	}
}

func selectEnvironmentKey(region string) string {
	parts := strings.Split(region, "-")
	switch parts[0] {
	case "cf":
		return "CF"
	case "k8s":
		return "KUBERNETES"
	case "neo":
		return "NEO"
	default:
		fmt.Printf("region %s does not fit any of the options, default CF is used\n", region)
		return "CF"
	}
}
