package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"gopkg.in/yaml.v3"
	coreV1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kcrConfigMapNamespace = "kcp-system"
	kcrNodemeterKey       = "nodemeterconfig.yaml"
)

type KCRVolumeProvider struct {
	k8sClient     client.Client
	configMapName string
}

func NewKCRVolumeProvider(k8sClient client.Client, configMapName string) *KCRVolumeProvider {
	return &KCRVolumeProvider{
		k8sClient:     k8sClient,
		configMapName: configMapName,
	}
}

// DefaultVolumeSizeGb reads the KCR ConfigMap on each call and returns the default volume size in GiB.
// Returns a TemporaryError for transient k8s errors (triggers RetryOperation).
// Returns a plain error if the machine type is not found (triggers OperationFailed).
func (p *KCRVolumeProvider) DefaultVolumeSizeGb(ctx context.Context, cloudProvider pkg.CloudProvider, machineType string) (int, error) {
	data, err := p.readAndParse(ctx)
	if err != nil {
		return 0, err
	}
	return lookupVolumeSize(data, cloudProvider, machineType)
}

// ValidateAllMachineTypes reads the ConfigMap once and verifies every machine type has a valid disk size entry.
// Used at startup; caller should fatalOnError on the result.
func (p *KCRVolumeProvider) ValidateAllMachineTypes(ctx context.Context, machines map[pkg.CloudProvider][]string) error {
	data, err := p.readAndParse(ctx)
	if err != nil {
		return err
	}
	var missing []string
	for cp, machineList := range machines {
		for _, mt := range machineList {
			if _, err := lookupVolumeSize(data, cp, mt); err != nil {
				missing = append(missing, fmt.Sprintf("%s/%s", cp, mt))
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing KCR volume sizes for: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (p *KCRVolumeProvider) readAndParse(ctx context.Context) (map[string]map[string]int, error) {
	cm := &coreV1.ConfigMap{}
	err := p.k8sClient.Get(ctx, client.ObjectKey{Namespace: kcrConfigMapNamespace, Name: p.configMapName}, cm)
	if err != nil {
		return nil, kebError.AsTemporaryError(err, "while reading KCR ConfigMap %s/%s", kcrConfigMapNamespace, p.configMapName)
	}
	nodemeterYAML, ok := cm.Data[kcrNodemeterKey]
	if !ok {
		return nil, fmt.Errorf("key %q missing from KCR ConfigMap %s/%s", kcrNodemeterKey, kcrConfigMapNamespace, p.configMapName)
	}
	return parseNodemeterYAML(nodemeterYAML)
}

func lookupVolumeSize(data map[string]map[string]int, cloudProvider pkg.CloudProvider, machineType string) (int, error) {
	providerKey := cloudProviderToKey(cloudProvider)
	providerData, ok := data[providerKey]
	if !ok {
		return 0, fmt.Errorf("provider %q not found in KCR ConfigMap", providerKey)
	}
	size, ok := providerData[strings.ToLower(machineType)]
	if !ok {
		return 0, fmt.Errorf("machine type %q not found in KCR ConfigMap for provider %q", machineType, providerKey)
	}
	if size == 0 {
		return 0, fmt.Errorf("machine type %q has zero or missing default_volume_size in KCR ConfigMap for provider %q", machineType, providerKey)
	}
	return size, nil
}

func cloudProviderToKey(cp pkg.CloudProvider) string {
	switch cp {
	case pkg.AWS:
		return "aws"
	case pkg.Azure:
		return "azure"
	case pkg.GCP:
		return "gcp"
	case pkg.SapConvergedCloud:
		return "openstack"
	case pkg.Alicloud:
		return "alicloud"
	default:
		return strings.ToLower(string(cp))
	}
}

type nodemeterDTO struct {
	Meters struct {
		Node struct {
			MachineTypes map[string]map[string]machineTypeEntryDTO `yaml:"machine_types"`
		} `yaml:"node"`
	} `yaml:"meters"`
}

type machineTypeEntryDTO struct {
	DefaultVolumeSize string `yaml:"default_volume_size"`
}

func parseNodemeterYAML(data string) (map[string]map[string]int, error) {
	var dto nodemeterDTO
	if err := yaml.Unmarshal([]byte(data), &dto); err != nil {
		return nil, fmt.Errorf("while parsing %s: %w", kcrNodemeterKey, err)
	}
	result := make(map[string]map[string]int)
	for providerName, machines := range dto.Meters.Node.MachineTypes {
		result[strings.ToLower(providerName)] = make(map[string]int)
		for machineName, entry := range machines {
			if entry.DefaultVolumeSize == "" {
				result[strings.ToLower(providerName)][strings.ToLower(machineName)] = 0
				continue
			}
			sizeStr := strings.TrimSuffix(entry.DefaultVolumeSize, "Gi")
			size, err := strconv.Atoi(sizeStr)
			if err != nil {
				return nil, fmt.Errorf("invalid default_volume_size %q for machine %s/%s: %w", entry.DefaultVolumeSize, providerName, machineName, err)
			}
			result[strings.ToLower(providerName)][strings.ToLower(machineName)] = size
		}
	}
	return result, nil
}
