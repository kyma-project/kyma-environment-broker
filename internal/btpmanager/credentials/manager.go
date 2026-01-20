package btpmgrcreds

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	apicorev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	BtpManagerSecretName      = "sap-btp-manager"
	BtpManagerSecretNamespace = "kyma-system"
)

var (
	BtpManagerLabels      = map[string]string{"app.kubernetes.io/managed-by": keb, "app.kubernetes.io/watched-by": keb}
	BtpManagerAnnotations = map[string]string{"Warning": "This secret is generated. Do not edit!"}
	KymaGvk               = schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "Kyma"}
)

const (
	keb                     = "kcp-kyma-environment-broker"
	kcpNamespace            = "kcp-system"
	instanceIdLabel         = "kyma-project.io/instance-id"
	skipReconciliationLabel = "kyma-project.io/skip-reconciliation"
)

const (
	secretClientId     = "clientid"
	secretClientSecret = "clientsecret"
	secretSmUrl        = "sm_url"
	secretTokenUrl     = "tokenurl"
	secretClusterId    = "cluster_id"
)

type K8sClientProvider interface {
	K8sClientForRuntimeID(rid string) (client.Client, error)
}

type Manager struct {
	ctx               context.Context
	instances         storage.Instances
	kcpK8sClient      client.Client
	dryRun            bool
	k8sClientProvider K8sClientProvider
	logger            *slog.Logger
}

func NewManager(ctx context.Context, kcpK8sClient client.Client, instanceDb storage.Instances, logs *slog.Logger, dryRun bool) *Manager {
	return &Manager{
		ctx:               ctx,
		instances:         instanceDb,
		kcpK8sClient:      kcpK8sClient,
		dryRun:            dryRun,
		logger:            logs,
		k8sClientProvider: kubeconfig.NewK8sClientFromSecretProvider(kcpK8sClient),
	}
}

func (s *Manager) MatchInstance(kymaName string) (*internal.Instance, error) {
	kyma := &unstructured.Unstructured{}
	kyma.SetGroupVersionKind(KymaGvk)
	err := s.kcpK8sClient.Get(s.ctx, client.ObjectKey{
		Namespace: kcpNamespace,
		Name:      kymaName,
	}, kyma)
	if err != nil && errors.IsNotFound(err) {
		return nil, fmt.Errorf("not found secret with name %s on cluster : %s", kymaName, err)
	} else if err != nil {
		return nil, fmt.Errorf("unexpected error while getting secret %s from cluster : %s", kymaName, err)
	}
	s.logger.Info(fmt.Sprintf("found kyma CR on kcp for kyma name: %s", kymaName))
	labels := kyma.GetLabels()
	instanceId, ok := labels[instanceIdLabel]
	if !ok {
		return nil, fmt.Errorf("not found instance for kyma name %s : %s", kymaName, err)
	}
	s.logger.Info(fmt.Sprintf("found instance id %s for kyma name %s", instanceId, kymaName))
	instance, err := s.instances.GetByID(instanceId)
	if err != nil {
		return nil, fmt.Errorf("while getting instance %s from db %s", instanceId, err)
	}
	s.logger.Info(fmt.Sprintf("instance %s found in db", instance.InstanceID))
	return instance, err
}

func (s *Manager) ReconcileAll(jobReconciliationDelay time.Duration, metrics *Metrics) (ReconcileStats, error) {
	instances, err := s.GetReconcileCandidates()
	if err != nil {
		return ReconcileStats{}, err
	}
	s.logger.Info(fmt.Sprintf("processing %d instances as candidates", len(instances)))

	updateDone, updateNotDoneDueError, updateNotDoneDueOkState, skippedCount := 0, 0, 0, 0
	for _, instance := range instances {
		time.Sleep(jobReconciliationDelay)
		updated, skipped, err := s.ReconcileSecretForInstance(&instance)
		if err != nil {
			s.logger.Error(fmt.Sprintf("while doing update, for instance: %s, %s", instance.InstanceID, err))
			updateNotDoneDueError++
			continue
		}
		s.updateMetrics(metrics, instance, skipped)
		if skipped {
			s.logger.Info(fmt.Sprintf("skipping instance %s", instance.InstanceID))
			skippedCount++
			continue
		}
		if updated {
			s.logger.Info(fmt.Sprintf("update done for instance %s", instance.InstanceID))
			updateDone++
		} else {
			s.logger.Info(fmt.Sprintf("no need to update instance %s", instance.InstanceID))
			updateNotDoneDueOkState++
		}
	}
	s.logger.Info(fmt.Sprintf("runtime-reconciler summary: total %d instances: %d skipped, %d are OK, update was needed (and done with success) for %d instances, errors occured for %d instances",
		len(instances), skippedCount, updateNotDoneDueOkState, updateDone, updateNotDoneDueError))
	return ReconcileStats{
		instanceCnt:     len(instances),
		updatedCnt:      updateDone,
		updateErrorsCnt: updateNotDoneDueError,
		skippedCnt:      skippedCount,
		notChangedCnt:   updateNotDoneDueOkState,
	}, nil
}

func (s *Manager) updateMetrics(metrics *Metrics, instance internal.Instance, runtimeSkipped bool) {
	if metrics != nil {
		if runtimeSkipped {
			metrics.skippedSecrets.With(prometheus.Labels{"shoot": instance.InstanceDetails.ShootName, "state": "skipped"}).Set(float64(1))
			metrics.skippedSecrets.With(prometheus.Labels{"shoot": instance.InstanceDetails.ShootName, "state": "reconciled"}).Set(float64(0))
		} else {
			metrics.skippedSecrets.With(prometheus.Labels{"shoot": instance.InstanceDetails.ShootName, "state": "reconciled"}).Set(float64(1))
			metrics.skippedSecrets.With(prometheus.Labels{"shoot": instance.InstanceDetails.ShootName, "state": "skipped"}).Set(float64(0))
		}
	}
}

func (s *Manager) GetReconcileCandidates() ([]internal.Instance, error) {
	allInstances, _, _, err := s.instances.List(dbmodel.InstanceFilter{})
	if err != nil {
		return nil, fmt.Errorf("while getting all instances %s", err)
	}
	s.logger.Info(fmt.Sprintf("total number of instances in db: %d", len(allInstances)))

	var instancesWithinRuntime []internal.Instance
	for _, instance := range allInstances {
		if !instance.Reconcilable {
			s.logger.Info(fmt.Sprintf("skipping instance %s because it is not reconcilable (no runtimeId, last op was deprovisioning or op is in progress)", instance.InstanceID))
			continue
		}

		if instance.Parameters.ErsContext.SMOperatorCredentials == nil || instance.InstanceDetails.ServiceManagerClusterID == "" {
			s.logger.Warn(fmt.Sprintf("skipping instance %s because there are no needed data attached to instance", instance.InstanceID))
			continue
		}

		instancesWithinRuntime = append(instancesWithinRuntime, instance)
		s.logger.Info(fmt.Sprintf("adding instance %s as candidate for reconciliation", instance.InstanceID))
	}

	s.logger.Info(fmt.Sprintf("from total number of instances (%d) took %d as candidates", len(allInstances), len(instancesWithinRuntime)))
	return instancesWithinRuntime, nil
}

func (s *Manager) ReconcileSecretForInstance(instance *internal.Instance) (bool, bool, error) {
	s.logger.Info(fmt.Sprintf("reconciliation of btp-manager secret started for instance %s", instance.InstanceID))

	futureSecret, err := PrepareSecret(instance.Parameters.ErsContext.SMOperatorCredentials, instance.InstanceDetails.ServiceManagerClusterID)
	if err != nil {
		return false, false, err
	}

	k8sClient, err := s.k8sClientProvider.K8sClientForRuntimeID(instance.RuntimeID)
	if err != nil {
		return false, false, fmt.Errorf("while getting k8sClient for instance %s : %w", instance.InstanceID, err)
	}
	s.logger.Info(fmt.Sprintf("connected to skr with success for instance %s", instance.InstanceID))

	currentSecret := &v1.Secret{}
	err = k8sClient.Get(context.Background(), client.ObjectKey{Name: BtpManagerSecretName, Namespace: BtpManagerSecretNamespace}, currentSecret)
	if err != nil && errors.IsNotFound(err) {
		s.logger.Info(fmt.Sprintf("sap-btp-manager secret for instance: %s not found on cluster", instance.InstanceID))
		if s.dryRun {
			s.logger.Info(fmt.Sprintf("[dry-run] secret for instance %s would be re-created", instance.InstanceID))
		} else {
			if err := CreateOrUpdateSecret(k8sClient, futureSecret, s.logger.With("instanceID", instance.InstanceID)); err != nil {
				return false, false, fmt.Errorf("while re-creating secret in cluster for instance %s", instance.InstanceID)
			}
			s.logger.Info(fmt.Sprintf("sap-btp-manager secret on cluster for instance %s re-created successfully", instance.InstanceID))
		}
		return true, false, nil
	} else if err != nil {
		return false, false, fmt.Errorf("while getting secret from cluster for instance %s : %s", instance.InstanceID, err)
	}

	if value, ok := currentSecret.Labels[skipReconciliationLabel]; ok && value == "true" {
		s.logger.Info(fmt.Sprintf("skip reconciliation of sap-btp-manager secret for instance %s", instance.InstanceID))
		return false, true, nil
	}

	notMatchingKeys, err := s.compareSecrets(currentSecret, futureSecret)
	if err != nil {
		return false, false, fmt.Errorf("validation of secrets failed with unexpected reason for instance: %s : %s", instance.InstanceID, err)
	} else if len(notMatchingKeys) > 0 {
		s.logger.Info(fmt.Sprintf("btp-manager secret on cluster does not match for instance credentials in db : %s, incorrect values for keys: %s", instance.InstanceID, strings.Join(notMatchingKeys, ",")))
		if s.dryRun {
			s.logger.Info(fmt.Sprintf("[dry-run] secret for instance %s would be updated", instance.InstanceID))
		} else {
			if err := CreateOrUpdateSecret(k8sClient, futureSecret, s.logger); err != nil {
				return false, false, fmt.Errorf("while updating secret in cluster for instance %s %s", instance.InstanceID, err)
			}
			s.logger.Info(fmt.Sprintf("btp-manager secret on cluster updated for instance %s to match state from instances db", instance.InstanceID))
		}
		return true, false, nil
	} else {
		s.logger.Info(fmt.Sprintf("instance %s OK: btp-manager secret on cluster match within expected data", instance.InstanceID))
	}

	return false, false, nil
}

func (s *Manager) compareSecrets(s1, s2 *v1.Secret) ([]string, error) {
	areSecretEqualByKey := func(key string) (bool, error) {
		currentValue, ok := s1.Data[key]
		if !ok {
			return false, fmt.Errorf("while getting the value for the key %s in the first secret", key)
		}
		expectedValue, ok := s2.Data[key]
		if !ok {
			return false, fmt.Errorf("while getting the value for the key %s in the second secret", key)
		}
		return reflect.DeepEqual(currentValue, expectedValue), nil
	}

	notEqual := make([]string, 0)
	for _, key := range []string{secretClientSecret, secretClientId, secretSmUrl, secretTokenUrl, secretClusterId} {
		equal, err := areSecretEqualByKey(key)
		if err != nil {
			return nil, fmt.Errorf("while comparing values for key %s : %s", key, err)
		}
		if !equal {
			notEqual = append(notEqual, key)
		}
	}

	return notEqual, nil
}

func getKubeConfigSecretName(runtimeId string) string {
	return fmt.Sprintf("kubeconfig-%s", runtimeId)
}

func PrepareSecret(credentials *internal.ServiceManagerOperatorCredentials, clusterID string) (*apicorev1.Secret, error) {
	if credentials == nil || clusterID == "" {
		return nil, fmt.Errorf("empty params given")
	}
	if credentials.ClientID == "" {
		return nil, fmt.Errorf("client Id not set")
	}
	if credentials.ClientSecret == "" {
		return nil, fmt.Errorf("clients ecret not set")
	}
	if credentials.ServiceManagerURL == "" {
		return nil, fmt.Errorf("service manager url not set")
	}
	if credentials.URL == "" {
		return nil, fmt.Errorf("url not set")
	}

	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        BtpManagerSecretName,
			Namespace:   BtpManagerSecretNamespace,
			Labels:      BtpManagerLabels,
			Annotations: BtpManagerAnnotations,
		},
		Data: map[string][]byte{
			secretClientId:     []byte(credentials.ClientID),
			secretClientSecret: []byte(credentials.ClientSecret),
			secretSmUrl:        []byte(credentials.ServiceManagerURL),
			secretTokenUrl:     []byte(credentials.URL),
			secretClusterId:    []byte(clusterID),
		},
		Type: apicorev1.SecretTypeOpaque,
	}, nil
}

func CreateOrUpdateSecret(k8sClient client.Client, futureSecret *apicorev1.Secret, log *slog.Logger) error {
	if futureSecret == nil {
		return fmt.Errorf("empty secret data given")
	}
	currentSecret := apicorev1.Secret{}
	getErr := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: BtpManagerSecretNamespace, Name: BtpManagerSecretName}, &currentSecret)
	switch {
	case getErr != nil && !apierrors.IsNotFound(getErr):
		return fmt.Errorf("failed to get the secret for BTP Manager: %s", getErr)
	case getErr != nil && apierrors.IsNotFound(getErr):
		namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: BtpManagerSecretNamespace}}
		createErr := k8sClient.Create(context.Background(), namespace)
		if createErr != nil && !apierrors.IsAlreadyExists(createErr) {
			return fmt.Errorf("could not create %s namespace: %s", BtpManagerSecretNamespace, createErr)
		}

		createErr = k8sClient.Create(context.Background(), futureSecret)
		if createErr != nil {
			return fmt.Errorf("failed to create the secret for BTP Manager: %s", createErr)
		}

		log.Info("the secret for BTP Manager created")
		return nil
	default:
		if !reflect.DeepEqual(currentSecret.Labels, BtpManagerLabels) {
			log.Warn(fmt.Sprintf("the secret %s was not created by KEB and its data will be overwritten", BtpManagerSecretName))
		}

		currentSecret.Data = futureSecret.Data
		currentSecret.ObjectMeta.Labels = futureSecret.ObjectMeta.Labels
		currentSecret.ObjectMeta.Annotations = futureSecret.ObjectMeta.Annotations
		updateErr := k8sClient.Update(context.Background(), &currentSecret)
		if updateErr != nil {
			return fmt.Errorf("failed to update the secret for BTP Manager: %s", updateErr)
		}

		log.Info("the secret for BTP Manager updated")
		return nil
	}
}
