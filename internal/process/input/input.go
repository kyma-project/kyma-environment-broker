package input

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/networking"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

const (
	trialSuffixLength    = 5
	maxRuntimeNameLength = 36
)

type Config struct {
	URL                           string
	ProvisioningTimeout           time.Duration          `envconfig:"default=6h"`
	DeprovisioningTimeout         time.Duration          `envconfig:"default=5h"`
	KubernetesVersion             string                 `envconfig:"default=1.16.9"`
	DefaultGardenerShootPurpose   string                 `envconfig:"default=development"`
	MachineImage                  string                 `envconfig:"optional"`
	MachineImageVersion           string                 `envconfig:"optional"`
	TrialNodesNumber              int                    `envconfig:"optional"`
	DefaultTrialProvider          internal.CloudProvider `envconfig:"default=Azure"`
	AutoUpdateKubernetesVersion   bool                   `envconfig:"default=false"`
	AutoUpdateMachineImageVersion bool                   `envconfig:"default=false"`
	MultiZoneCluster              bool                   `envconfig:"default=false"`
	ControlPlaneFailureTolerance  string                 `envconfig:"optional"`
	GardenerClusterStepTimeout    time.Duration          `envconfig:"default=3m"`
	RuntimeResourceStepTimeout    time.Duration          `envconfig:"default=8m"`
	ClusterUpdateStepTimeout      time.Duration          `envconfig:"default=2h"`
	EnableShootAndSeedSameRegion  bool                   `envconfig:"default=false"`
}

type RuntimeInput struct {
	muLabels sync.Mutex

	provisionRuntimeInput gqlschema.ProvisionRuntimeInput
	upgradeRuntimeInput   gqlschema.UpgradeRuntimeInput
	upgradeShootInput     gqlschema.UpgradeShootInput
	labels                map[string]string

	config                   *internal.ConfigForPlan
	hyperscalerInputProvider HyperscalerInputProvider
	provisioningParameters   internal.ProvisioningParameters
	shootName                *string

	oidcDefaultValues internal.OIDCConfigDTO
	oidcLastValues    gqlschema.OIDCConfigInput

	trialNodesNumber             int
	instanceID                   string
	runtimeID                    string
	kubeconfig                   string
	shootDomain                  string
	shootDnsProviders            gardener.DNSProvidersData
	clusterName                  string
	enableShootAndSeedSameRegion bool
}

func (r *RuntimeInput) Configuration() *internal.ConfigForPlan {
	return r.config
}

func (r *RuntimeInput) SetProvisioningParameters(params internal.ProvisioningParameters) internal.ProvisionerInputCreator {
	r.provisioningParameters = params
	return r
}

func (r *RuntimeInput) SetShootName(name string) internal.ProvisionerInputCreator {
	r.shootName = &name
	return r
}

func (r *RuntimeInput) SetShootDomain(name string) internal.ProvisionerInputCreator {
	r.shootDomain = name
	return r
}

func (r *RuntimeInput) SetShootDNSProviders(dnsProviders gardener.DNSProvidersData) internal.ProvisionerInputCreator {
	r.shootDnsProviders = dnsProviders
	return r
}

func (r *RuntimeInput) SetInstanceID(instanceID string) internal.ProvisionerInputCreator {
	r.instanceID = instanceID
	return r
}

func (r *RuntimeInput) SetRuntimeID(runtimeID string) internal.ProvisionerInputCreator {
	r.runtimeID = runtimeID
	return r
}

func (r *RuntimeInput) SetKubeconfig(kubeconfig string) internal.ProvisionerInputCreator {
	r.kubeconfig = kubeconfig
	return r
}

func (r *RuntimeInput) SetClusterName(name string) internal.ProvisionerInputCreator {
	if name != "" {
		r.clusterName = name
	}
	return r
}

func (r *RuntimeInput) SetOIDCLastValues(oidcConfig gqlschema.OIDCConfigInput) internal.ProvisionerInputCreator {
	r.oidcLastValues = oidcConfig
	return r
}

func (r *RuntimeInput) SetLabel(key, value string) internal.ProvisionerInputCreator {
	r.muLabels.Lock()
	defer r.muLabels.Unlock()

	if r.provisionRuntimeInput.RuntimeInput.Labels == nil {
		r.provisionRuntimeInput.RuntimeInput.Labels = gqlschema.Labels{}
	}

	(r.provisionRuntimeInput.RuntimeInput.Labels)[key] = value
	return r
}

func (r *RuntimeInput) CreateProvisionRuntimeInput() (gqlschema.ProvisionRuntimeInput, error) {
	for _, step := range []struct {
		name    string
		execute func() error
	}{
		{
			name:    "applying provisioning parameters customization",
			execute: r.applyProvisioningParametersForProvisionRuntime,
		},
		{
			name:    "applying global configuration",
			execute: r.applyGlobalConfigurationForProvisionRuntime,
		},
		{
			name:    "removing forbidden chars and adding random string to runtime name",
			execute: r.adjustRuntimeName,
		},
		{
			name:    "set number of nodes from configuration",
			execute: r.setNodesForTrialProvision,
		},
		{
			name:    "configure OIDC",
			execute: r.configureOIDC,
		},
		{
			name:    "configure DNS",
			execute: r.configureDNS,
		},
		{
			name:    "configure networking",
			execute: r.configureNetworking,
		},
	} {
		if err := step.execute(); err != nil {
			return gqlschema.ProvisionRuntimeInput{}, fmt.Errorf("while %s: %w", step.name, err)
		}
	}

	return r.provisionRuntimeInput, nil
}

func (r *RuntimeInput) CreateUpgradeRuntimeInput() (gqlschema.UpgradeRuntimeInput, error) {
	for _, step := range []struct {
		name    string
		execute func() error
	}{
		{
			name:    "applying global configuration",
			execute: r.applyGlobalConfigurationForUpgradeRuntime,
		},
		{
			name:    "set number of nodes from configuration",
			execute: r.setNodesForTrialProvision,
		},
	} {
		if err := step.execute(); err != nil {
			return gqlschema.UpgradeRuntimeInput{}, fmt.Errorf("while %s: %w", step.name, err)
		}
	}

	return r.upgradeRuntimeInput, nil
}

func (r *RuntimeInput) CreateUpgradeShootInput() (gqlschema.UpgradeShootInput, error) {

	for _, step := range []struct {
		name    string
		execute func() error
	}{
		{
			name:    "applying provisioning parameters customization",
			execute: r.applyProvisioningParametersForUpgradeShoot,
		},
		{
			name:    "setting number of trial nodes from configuration",
			execute: r.setNodesForTrialUpgrade,
		},
		{
			name:    "configure OIDC",
			execute: r.configureOIDC,
		},
	} {
		if err := step.execute(); err != nil {
			return gqlschema.UpgradeShootInput{}, fmt.Errorf("while %s: %w", step.name, err)
		}
	}
	return r.upgradeShootInput, nil
}

func (r *RuntimeInput) Provider() internal.CloudProvider {
	return r.hyperscalerInputProvider.Provider()
}

func emptyIfNil(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func falseIfNil(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

func (r *RuntimeInput) CreateProvisionClusterInput() (gqlschema.ProvisionRuntimeInput, error) {
	result, err := r.CreateProvisionRuntimeInput()
	if err != nil {
		return gqlschema.ProvisionRuntimeInput{}, nil
	}
	result.KymaConfig = nil
	return result, nil
}

func (r *RuntimeInput) applyProvisioningParametersForProvisionRuntime() error {
	params := r.provisioningParameters.Parameters
	updateString(&r.provisionRuntimeInput.RuntimeInput.Name, &params.Name)

	if r.provisioningParameters.PlanID == broker.OwnClusterPlanID {
		return nil
	}

	if r.enableShootAndSeedSameRegion {
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.ShootAndSeedSameRegion = params.ShootAndSeedSameRegion
	}
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.MaxUnavailable, params.MaxUnavailable)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.MaxSurge, params.MaxSurge)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMin, params.AutoScalerMin)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMax, params.AutoScalerMax)
	updateInt(r.provisionRuntimeInput.ClusterConfig.GardenerConfig.VolumeSizeGb, params.VolumeSizeGb)
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.Name, r.shootName)
	if params.Region != nil && *params.Region != "" {
		updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.Region, params.Region)
	}
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.MachineType, params.MachineType)
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.TargetSecret, params.TargetSecret)
	updateString(r.provisionRuntimeInput.ClusterConfig.GardenerConfig.Purpose, params.Purpose)
	if params.LicenceType != nil {
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.LicenceType = params.LicenceType
	}

	// admins parameter check
	if len(r.provisioningParameters.Parameters.RuntimeAdministrators) == 0 {
		// default admin set from UserID in ERSContext
		r.provisionRuntimeInput.ClusterConfig.Administrators = []string{r.provisioningParameters.ErsContext.UserID}
	} else {
		// set admins for new runtime
		r.provisionRuntimeInput.ClusterConfig.Administrators = []string{}
		r.provisionRuntimeInput.ClusterConfig.Administrators = append(
			r.provisionRuntimeInput.ClusterConfig.Administrators,
			r.provisioningParameters.Parameters.RuntimeAdministrators...,
		)
	}

	r.hyperscalerInputProvider.ApplyParameters(r.provisionRuntimeInput.ClusterConfig, r.provisioningParameters)

	return nil
}

func (r *RuntimeInput) applyProvisioningParametersForUpgradeShoot() error {
	if len(r.provisioningParameters.Parameters.RuntimeAdministrators) != 0 {
		// prepare new admins list for existing runtime
		newAdministrators := make([]string, 0, len(r.provisioningParameters.Parameters.RuntimeAdministrators))
		newAdministrators = append(newAdministrators, r.provisioningParameters.Parameters.RuntimeAdministrators...)
		r.upgradeShootInput.Administrators = newAdministrators
	} else {
		if r.provisioningParameters.ErsContext.UserID != "" {
			// get default admin (user_id from provisioning operation)
			r.upgradeShootInput.Administrators = []string{r.provisioningParameters.ErsContext.UserID}
		} else {
			// some old clusters does not have an user_id
			r.upgradeShootInput.Administrators = []string{}
		}
	}

	updateInt(r.upgradeShootInput.GardenerConfig.MaxSurge, r.provisioningParameters.Parameters.MaxSurge)
	updateInt(r.upgradeShootInput.GardenerConfig.MaxUnavailable, r.provisioningParameters.Parameters.MaxUnavailable)

	return nil
}

func (r *RuntimeInput) applyGlobalConfigurationForProvisionRuntime() error {
	strategy := gqlschema.ConflictStrategyReplace
	r.provisionRuntimeInput.KymaConfig.ConflictStrategy = &strategy
	return nil
}

func (r *RuntimeInput) applyGlobalConfigurationForUpgradeRuntime() error {
	strategy := gqlschema.ConflictStrategyReplace
	r.upgradeRuntimeInput.KymaConfig.ConflictStrategy = &strategy
	return nil
}

func (r *RuntimeInput) adjustRuntimeName() error {
	// if the cluster name was created before, it must be used instead of generating one
	if r.clusterName != "" {
		r.provisionRuntimeInput.RuntimeInput.Name = r.clusterName
		return nil
	}

	reg, err := regexp.Compile("[^a-zA-Z0-9\\-\\.]+")
	if err != nil {
		return fmt.Errorf("while compiling regexp: %w", err)
	}

	name := strings.ToLower(reg.ReplaceAllString(r.provisionRuntimeInput.RuntimeInput.Name, ""))
	modifiedLength := len(name) + trialSuffixLength + 1
	if modifiedLength > maxRuntimeNameLength {
		name = trimLastCharacters(name, modifiedLength-maxRuntimeNameLength)
	}

	r.provisionRuntimeInput.RuntimeInput.Name = fmt.Sprintf("%s-%s", name, randomString(trialSuffixLength))
	return nil
}

func (r *RuntimeInput) configureDNS() error {
	dnsParamsToSet := gqlschema.DNSConfigInput{}

	// if dns providers is given
	if len(r.shootDnsProviders.Providers) != 0 {
		for _, v := range r.shootDnsProviders.Providers {
			dnsParamsToSet.Providers = append(dnsParamsToSet.Providers, &gqlschema.DNSProviderInput{
				DomainsInclude: v.DomainsInclude,
				Primary:        v.Primary,
				SecretName:     v.SecretName,
				Type:           v.Type,
			})
		}
	}

	dnsParamsToSet.Domain = r.shootDomain

	if r.provisionRuntimeInput.ClusterConfig != nil &&
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig != nil {
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.DNSConfig = &dnsParamsToSet
	}

	return nil
}

func (r *RuntimeInput) configureOIDC() error {
	// set default or provided params to provisioning/update input (if exists)
	// This method could be used for:
	// provisioning (upgradeShootInput.GardenerConfig is nil)
	// or upgrade (provisionRuntimeInput.ClusterConfig is nil)

	if r.provisionRuntimeInput.ClusterConfig != nil {
		oidcParamsToSet := r.setOIDCForProvisioning()
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.OidcConfig = oidcParamsToSet
	}
	if r.upgradeShootInput.GardenerConfig != nil {
		oidcParamsToSet := r.setOIDCForUpgrade()
		r.upgradeShootInput.GardenerConfig.OidcConfig = oidcParamsToSet
	}
	return nil
}

func (r *RuntimeInput) configureNetworking() error {
	if r.provisioningParameters.Parameters.Networking == nil {
		return nil
	}
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.WorkerCidr,
		&r.provisioningParameters.Parameters.Networking.NodesCidr)

	// if the Networking section is set, then
	r.provisionRuntimeInput.ClusterConfig.GardenerConfig.PodsCidr = ptr.String(networking.DefaultPodsCIDR)
	updateString(r.provisionRuntimeInput.ClusterConfig.GardenerConfig.PodsCidr,
		r.provisioningParameters.Parameters.Networking.PodsCidr)

	r.provisionRuntimeInput.ClusterConfig.GardenerConfig.ServicesCidr = ptr.String(networking.DefaultServicesCIDR)
	updateString(r.provisionRuntimeInput.ClusterConfig.GardenerConfig.ServicesCidr,
		r.provisioningParameters.Parameters.Networking.ServicesCidr)

	return nil
}

func (r *RuntimeInput) setNodesForTrialProvision() error {
	// parameter with number of notes for trial plan is optional; if parameter is not set value is equal to 0
	if r.trialNodesNumber == 0 {
		return nil
	}
	if broker.IsTrialPlan(r.provisioningParameters.PlanID) {
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMin = r.trialNodesNumber
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMax = r.trialNodesNumber
	}
	return nil
}

func (r *RuntimeInput) setNodesForTrialUpgrade() error {
	// parameter with number of nodes for trial plan is optional; if parameter is not set value is equal to 0
	if r.trialNodesNumber == 0 {
		return nil
	}
	if broker.IsTrialPlan(r.provisioningParameters.PlanID) {
		r.upgradeShootInput.GardenerConfig.AutoScalerMin = &r.trialNodesNumber
		r.upgradeShootInput.GardenerConfig.AutoScalerMax = &r.trialNodesNumber
	}
	return nil
}

func (r *RuntimeInput) setOIDCForProvisioning() *gqlschema.OIDCConfigInput {
	oidcConfig := &gqlschema.OIDCConfigInput{
		ClientID:       r.oidcDefaultValues.ClientID,
		GroupsClaim:    r.oidcDefaultValues.GroupsClaim,
		IssuerURL:      r.oidcDefaultValues.IssuerURL,
		SigningAlgs:    r.oidcDefaultValues.SigningAlgs,
		UsernameClaim:  r.oidcDefaultValues.UsernameClaim,
		UsernamePrefix: r.oidcDefaultValues.UsernamePrefix,
	}

	if r.provisioningParameters.Parameters.OIDC.IsProvided() {
		r.setOIDCFromProvisioningParameters(oidcConfig)
	}

	return oidcConfig
}

func (r *RuntimeInput) setOIDCForUpgrade() *gqlschema.OIDCConfigInput {
	oidcConfig := r.oidcLastValues
	r.setOIDCDefaultValuesIfEmpty(&oidcConfig)

	if r.provisioningParameters.Parameters.OIDC.IsProvided() {
		r.setOIDCFromProvisioningParameters(&oidcConfig)
	}

	return &oidcConfig
}

func (r *RuntimeInput) setOIDCFromProvisioningParameters(oidcConfig *gqlschema.OIDCConfigInput) {
	providedOIDC := r.provisioningParameters.Parameters.OIDC
	oidcConfig.ClientID = providedOIDC.ClientID
	oidcConfig.IssuerURL = providedOIDC.IssuerURL
	if len(providedOIDC.GroupsClaim) != 0 {
		oidcConfig.GroupsClaim = providedOIDC.GroupsClaim
	}
	if len(providedOIDC.SigningAlgs) != 0 {
		oidcConfig.SigningAlgs = providedOIDC.SigningAlgs
	}
	if len(providedOIDC.UsernameClaim) != 0 {
		oidcConfig.UsernameClaim = providedOIDC.UsernameClaim
	}
	if len(providedOIDC.UsernamePrefix) != 0 {
		oidcConfig.UsernamePrefix = providedOIDC.UsernamePrefix
	}
}

func (r *RuntimeInput) setOIDCDefaultValuesIfEmpty(oidcConfig *gqlschema.OIDCConfigInput) {
	if oidcConfig.ClientID == "" {
		oidcConfig.ClientID = r.oidcDefaultValues.ClientID
	}
	if oidcConfig.IssuerURL == "" {
		oidcConfig.IssuerURL = r.oidcDefaultValues.IssuerURL
	}
	if oidcConfig.GroupsClaim == "" {
		oidcConfig.GroupsClaim = r.oidcDefaultValues.GroupsClaim
	}
	if len(oidcConfig.SigningAlgs) == 0 {
		oidcConfig.SigningAlgs = r.oidcDefaultValues.SigningAlgs
	}
	if oidcConfig.UsernameClaim == "" {
		oidcConfig.UsernameClaim = r.oidcDefaultValues.UsernameClaim
	}
	if oidcConfig.UsernamePrefix == "" {
		oidcConfig.UsernamePrefix = r.oidcDefaultValues.UsernamePrefix
	}
}

func updateString(toUpdate *string, value *string) {
	if value != nil {
		*toUpdate = *value
	}
}

func updateBool(toUpdate *bool, value *bool) {
	if value != nil {
		toUpdate = value
	}
}

func updateInt(toUpdate *int, value *int) {
	if value != nil {
		*toUpdate = *value
	}
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func trimLastCharacters(s string, count int) string {
	s = s[:len(s)-count]
	return s
}

func resolveValueType(v interface{}) interface{} {
	// this is a workaround. Finally we have to obtain the type during the reading overrides
	var val interface{}
	switch v {
	case "true":
		val = true
	case "false":
		val = false
	default:
		val = v
	}

	return val
}
