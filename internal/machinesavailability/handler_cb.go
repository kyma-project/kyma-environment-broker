package machinesavailability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/subscriptions"
)

type HandlerCB struct {
	providerSpec   *configuration.ProviderSpec
	rulesService   *rules.RulesService
	gardenerClient *gardener.Client
	clientFactory  aws.ClientFactory
	logger         *slog.Logger
}

func NewHandlerCB(
	providerSpec *configuration.ProviderSpec,
	rulesService *rules.RulesService,
	gardenerClient *gardener.Client,
	clientFactory aws.ClientFactory,
	logger *slog.Logger,
) *HandlerCB {
	return &HandlerCB{
		providerSpec:   providerSpec,
		rulesService:   rulesService,
		gardenerClient: gardenerClient,
		clientFactory:  clientFactory,
		logger:         logger.With("service", "MachinesAvailabilityHandler"),
	}
}

func (h *HandlerCB) AttachRoutes(router *httputil.Router) {
	router.HandleFunc(machinesAvailabilityPath, h.getMachinesAvailability)
}

func (h *HandlerCB) getMachinesAvailability(w http.ResponseWriter, req *http.Request) {
	supportedProviders := []runtime.CloudProvider{runtime.AWS}
	var providersData ProvidersData

	for _, provider := range supportedProviders {
		providerEntry := Provider{
			Name:         provider,
			MachineTypes: []MachineType{},
		}

		regionSupportingMachine, err := h.providerSpec.RegionSupportingMachine(string(provider))
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		accessKeyID, secretAccessKey, err := h.clientCredentials(strings.ToLower(string(provider)))
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		machineTypes := h.providerSpec.MachineTypes(provider)
		machineFamilies := make(map[string]string)
		for _, machineType := range machineTypes {
			var family string
			if provider == runtime.AWS {
				// For AWS, machine types follow the pattern "<family>.<size>".
				parts := strings.SplitN(machineType, ".", 2)
				family = parts[0]
			} else {
				httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("%s provider not supported", provider))
				return
			}
			machineFamilies[family] = machineType
		}

		for machineFamily, machineType := range machineFamilies {
			machineTypeEntry := MachineType{
				Name:    machineFamily,
				Regions: []Region{},
			}

			regions := regionSupportingMachine.SupportedRegions(machineType)
			if len(regions) == 0 {
				regions = h.providerSpec.Regions(provider)
			}

			for _, region := range regions {
				client, err := h.clientFactory.New(context.Background(), accessKeyID, secretAccessKey, region)
				if err != nil {
					httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
					return
				}

				count, err := client.AvailableZonesCount(context.Background(), machineType)
				if err != nil {
					httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
					return
				}

				highAvailability := count >= highAvailabilityThreshold
				machineTypeEntry.Regions = append(machineTypeEntry.Regions, Region{
					Name:             region,
					HighAvailability: highAvailability,
				})
			}

			providerEntry.MachineTypes = append(providerEntry.MachineTypes, machineTypeEntry)
		}

		sort.Slice(providerEntry.MachineTypes, func(i, j int) bool {
			return providerEntry.MachineTypes[i].Name < providerEntry.MachineTypes[j].Name
		})

		providersData.Providers = append(providersData.Providers, providerEntry)
	}

	httputil.WriteResponse(w, http.StatusOK, providersData)
}

func (h *HandlerCB) clientCredentials(provider string) (string, string, error) {
	matchedRule, err := h.matchRule(provider)
	if err != nil {
		return "", "", err
	}

	credentialsBinding, err := h.getCredentialsBindingForRule(matchedRule)
	if err != nil {
		return "", "", err
	}

	h.logger.Info(fmt.Sprintf("getting subscription secret with name %s/%s", credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName()))
	secret, err := h.gardenerClient.GetSecret(credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName())
	if err != nil {
		return "", "", fmt.Errorf("unable to get secret %s/%s", credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName())
	}

	accessKeyID, secretAccessKey, err := aws.ExtractCredentials(secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to extract AWS credentials")
	}
	return accessKeyID, secretAccessKey, nil
}

func (h *HandlerCB) matchRule(provider string) (rules.Result, error) {
	attr := &rules.ProvisioningAttributes{
		Plan:        provider,
		Hyperscaler: provider,
	}

	matchedRule, found := h.rulesService.MatchProvisioningAttributesWithValidRuleset(attr)
	if !found {
		return rules.Result{}, fmt.Errorf("no matching rule for provisioning attributes %q", attr)
	}

	h.logger.Info(fmt.Sprintf("matched rule: %q", matchedRule.Rule()))
	return matchedRule, nil
}

func (h *HandlerCB) getCredentialsBindingForRule(matchedRule rules.Result) (*gardener.CredentialsBinding, error) {
	labelSelectorBuilder := subscriptions.NewLabelSelectorFromRuleset(matchedRule)
	labelSelector := labelSelectorBuilder.BuildAnySubscription()

	h.logger.Info(fmt.Sprintf("getting secret binding with selector %q", labelSelector))
	credentialsBindings, err := h.gardenerClient.GetCredentialsBindings(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("while getting secret bindings with selector %q: %w", labelSelector, err)
	}
	if credentialsBindings == nil || len(credentialsBindings.Items) == 0 {
		return nil, fmt.Errorf("no credentials bindings found for selector %q", labelSelector)
	}

	return gardener.NewCredentialsBinding(credentialsBindings.Items[0]), nil
}
