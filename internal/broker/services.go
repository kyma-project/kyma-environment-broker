package broker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kyma-project/kyma-environment-broker/internal/assuredworkloads"

	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"

	"github.com/kyma-project/kyma-environment-broker/internal/middleware"

	"github.com/pivotal-cf/brokerapi/v8/domain"
)

const (
	ControlsOrderKey = "_controlsOrder"
	PropertiesKey    = "properties"
)

type ServicesEndpoint struct {
	log            *slog.Logger
	cfg            Config
	servicesConfig ServicesConfig

	enabledPlanIDs                map[string]struct{}
	convergedCloudRegionsProvider ConvergedCloudRegionProvider
}

func NewServices(cfg Config, servicesConfig ServicesConfig, log *slog.Logger, convergedCloudRegionsProvider ConvergedCloudRegionProvider) *ServicesEndpoint {
	enabledPlanIDs := map[string]struct{}{}
	for _, planName := range cfg.EnablePlans {
		id := PlanIDsMapping[planName]
		enabledPlanIDs[id] = struct{}{}
	}

	return &ServicesEndpoint{
		log:                           log.With("service", "ServicesEndpoint"),
		cfg:                           cfg,
		servicesConfig:                servicesConfig,
		enabledPlanIDs:                enabledPlanIDs,
		convergedCloudRegionsProvider: convergedCloudRegionsProvider,
	}
}

// Services gets the catalog of services offered by the service broker
//
//	GET /v2/catalog
func (b *ServicesEndpoint) Services(ctx context.Context) ([]domain.Service, error) {
	var availableServicePlans []domain.ServicePlan
	bindable := true
	// we scope to the kymaruntime service only
	class, ok := b.servicesConfig[KymaServiceName]
	if !ok {
		return nil, fmt.Errorf("while getting %s class data", KymaServiceName)
	}

	provider, ok := middleware.ProviderFromContext(ctx)
	platformRegion, ok := middleware.RegionFromContext(ctx)
	for _, plan := range Plans(class.Plans,
		provider,
		b.cfg.IncludeAdditionalParamsInSchema,
		euaccess.IsEURestrictedAccess(platformRegion),
		b.cfg.UseSmallerMachineTypes,
		b.cfg.EnableShootAndSeedSameRegion,
		b.convergedCloudRegionsProvider.GetRegions(platformRegion),
		assuredworkloads.IsKSA(platformRegion),
	) {

		// filter out not enabled plans
		if _, exists := b.enabledPlanIDs[plan.ID]; !exists {
			continue
		}
		if b.cfg.Binding.Enabled && b.cfg.Binding.BindablePlans.Contains(plan.Name) {
			plan.Bindable = &bindable
		}

		availableServicePlans = append(availableServicePlans, plan)
	}

	return []domain.Service{
		{
			ID:                   KymaServiceID,
			Name:                 KymaServiceName,
			Description:          class.Description,
			Bindable:             false,
			InstancesRetrievable: true,
			Tags: []string{
				"SAP",
				"Kyma",
			},
			Plans: availableServicePlans,
			Metadata: &domain.ServiceMetadata{
				DisplayName:         class.Metadata.DisplayName,
				ImageUrl:            class.Metadata.ImageUrl,
				LongDescription:     class.Metadata.LongDescription,
				ProviderDisplayName: class.Metadata.ProviderDisplayName,
				DocumentationUrl:    class.Metadata.DocumentationUrl,
				SupportUrl:          class.Metadata.SupportUrl,
			},
			AllowContextUpdates: true,
		},
	}, nil
}
