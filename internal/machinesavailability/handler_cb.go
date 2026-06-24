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
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/subscriptions"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	machinesAvailabilityPath  = "/oauth/v2/machines_availability"
	highAvailabilityThreshold = 3
)

type ProvidersData struct {
	Providers []Provider `json:"providers"`
}

type Provider struct {
	Name         runtime.CloudProvider `json:"name"`
	MachineTypes []MachineType         `json:"machine_types"`
}

type MachineType struct {
	Name    string   `json:"name"`
	Regions []Region `json:"regions"`
}

type Region struct {
	Name             string `json:"name"`
	HighAvailability bool   `json:"high_availability"`
}

type HandlerCB struct {
	providerSpec   *configuration.ProviderSpec
	rulesService   *rules.RulesService
	gardenerClient *gardener.Client
	factory        hyperscalers.Factory
	logger         *slog.Logger
}

func NewHandlerCB(
	providerSpec *configuration.ProviderSpec,
	rulesService *rules.RulesService,
	gardenerClient *gardener.Client,
	factory hyperscalers.Factory,
	logger *slog.Logger,
) *HandlerCB {
	return &HandlerCB{
		providerSpec:   providerSpec,
		rulesService:   rulesService,
		gardenerClient: gardenerClient,
		factory:        factory,
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

		secret, err := h.getSecret(strings.ToLower(string(provider)))
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		machineTypes := h.providerSpec.MachineTypes(provider)
		machineFamilies := make(map[string]string)
		for _, machineType := range machineTypes {
			family, ok := h.providerSpec.MachineFamily(provider, machineType)
			if !ok {
				httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("machine family extraction not supported for provider %s", provider))
				return
			}
			machineFamilies[family] = machineType
		}

		for machineFamily, machineType := range machineFamilies {
			machineTypeEntry := MachineType{
				Name:    machineFamily,
				Regions: []Region{},
			}

			regions := h.providerSpec.SupportedRegions(provider, machineType)
			if len(regions) == 0 {
				regions = h.providerSpec.Regions(provider)
			}

			for _, region := range regions {
				client, err := h.factory.NewFromSecret(context.Background(), provider, secret, region)
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

func (h *HandlerCB) getSecret(provider string) (*unstructured.Unstructured, error) {
	matchedRule, err := h.matchRule(provider)
	if err != nil {
		return nil, err
	}

	credentialsBinding, err := h.getCredentialsBindingForRule(matchedRule)
	if err != nil {
		return nil, err
	}

	h.logger.Info(fmt.Sprintf("getting subscription secret with name %s/%s", credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName()))
	secret, err := h.gardenerClient.GetSecret(credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName())
	if err != nil {
		return nil, fmt.Errorf("unable to get secret %s/%s: %w", credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName(), err)
	}
	return secret, nil
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
