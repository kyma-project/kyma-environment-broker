package broker

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/kyma-project/kyma-environment-broker/internal/middleware"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

const (
	ControlsOrderKey = "_controlsOrder"
	PropertiesKey    = "properties"
)

type ServicesEndpoint struct {
	log            *slog.Logger
	cfg            Config
	servicesConfig ServicesConfig

	enabledPlanIDs              map[string]struct{}
	defaultOIDCConfig           *pkg.OIDCConfigDTO
	useSmallerMachineTypes      bool
	ingressFilteringFeatureFlag bool
	ingressFilteringPlans       StringList
	schemaService               *SchemaService
}

func NewServices(cfg Config, schemaService *SchemaService, servicesConfig ServicesConfig, log *slog.Logger, defaultOIDCConfig pkg.OIDCConfigDTO, imConfig InfrastructureManager) *ServicesEndpoint {
	enabledPlanIDs := map[string]struct{}{}
	for _, planName := range cfg.EnablePlans {
		id := PlanIDsMapping[planName]
		enabledPlanIDs[id] = struct{}{}
	}

	return &ServicesEndpoint{
		log:                    log.With("service", "ServicesEndpoint"),
		cfg:                    cfg,
		servicesConfig:         servicesConfig,
		enabledPlanIDs:         enabledPlanIDs,
		defaultOIDCConfig:      &defaultOIDCConfig,
		useSmallerMachineTypes: imConfig.UseSmallerMachineTypes,
		ingressFilteringPlans:  imConfig.IngressFilteringPlans,
		schemaService:          schemaService,
	}
}

// Services gets the catalog of services offered by the service broker
//
//	GET /v2/catalog
func (se *ServicesEndpoint) Services(ctx context.Context) ([]domain.Service, error) {
	var availableServicePlans []domain.ServicePlan
	bindable := true
	// we scope to the kymaruntime service only
	class, ok := se.servicesConfig[KymaServiceName]
	if !ok {
		return nil, fmt.Errorf("while getting %s class data", KymaServiceName)
	}

	provider, ok := middleware.ProviderFromContext(ctx)
	platformRegion, ok := middleware.RegionFromContext(ctx)

	for _, plan := range se.schemaService.Plans(class.Plans, platformRegion, provider) {
		// filter out not enabled plans
		if _, exists := se.enabledPlanIDs[plan.ID]; !exists {
			continue
		}

		if se.cfg.Binding.Enabled && se.cfg.Binding.BindablePlans.Contains(plan.Name) {
			plan.Bindable = &bindable
		}
		availableServicePlans = append(availableServicePlans, plan)
	}

	sort.Slice(availableServicePlans, func(i, j int) bool {
		return availableServicePlans[i].Name < availableServicePlans[j].Name
	})

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
