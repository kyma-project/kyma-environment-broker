package btpmgrcreds

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/apimachinery/pkg/api/errors"

	uuid2 "github.com/google/uuid"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apicorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	expectedInstancesCount         = 3
	expectedRejectedInstancesCount = 1
	expectedAllInstancesCount      = expectedInstancesCount + expectedRejectedInstancesCount
	credentialsLen                 = 16
	jobReconciliationDelay         = time.Second * 0
)

var (
	changedInstancesCount = int(math.Ceil(expectedInstancesCount / 2))
	testDataIndexes       = []int{0, 2}
	random                = rand.New(rand.NewSource(1))
)

type Environment struct {
	ctx           context.Context
	skrs          []*envtest.Environment
	instanceIds   []string
	kcp           client.Client
	brokerStorage storage.BrokerStorage
	logs          *slog.Logger
	manager       *Manager
	job           *Job
	t             *testing.T
}

func InitEnvironment(ctx context.Context, t *testing.T) *Environment {
	logs := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	newEnvironment := &Environment{
		skrs:          make([]*envtest.Environment, 0),
		brokerStorage: storage.NewMemoryStorage(),
		logs:          logs,
		ctx:           ctx,
		t:             t,
	}

	newEnvironment.createTestData()
	newEnvironment.manager = NewManager(ctx, newEnvironment.kcp, newEnvironment.brokerStorage.Instances(), logs, false)
	newEnvironment.job = NewJob(newEnvironment.manager, logs, prometheus.NewRegistry(), "8081", "runtime-reconciler-test")
	newEnvironment.assertNumberOfInstancesInDb(expectedAllInstancesCount)
	return newEnvironment
}

func TestBtpManagerReconciler(t *testing.T) {
	pid := internal.SetupEnvtest(t)
	defer func() {
		internal.CleanupEnvtestBinaries(pid)
	}()

	t.Run("btp manager credentials tests", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		environment := InitEnvironment(ctx, t)

		t.Run("reconcile when all secrets are not set", func(t *testing.T) {
			environment.assertAllSecretsNotExists()
			stats, err := environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)
			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, expectedInstancesCount, stats.updatedCnt)
			assert.Equal(t, 0, stats.updateErrorsCnt+stats.notChangedCnt)
			assert.Zero(t, stats.skippedCnt)
			environment.assertAllSecretDataAreSet()
			environment.assureConsistency()
		})

		// TODO all following test cases depend on the previous one - remove this dependency
		t.Run("reconcile when all secrets are correct", func(t *testing.T) {
			environment.assertAllSecretDataAreSet()
			stats, err := environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)
			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)
			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, 0, stats.updatedCnt)
			assert.Equal(t, expectedInstancesCount, stats.updateErrorsCnt+stats.notChangedCnt)
			assert.Zero(t, stats.skippedCnt)
			environment.assertAllSecretDataAreSet()
			environment.assureConsistency()
		})

		t.Run("reconcile when some secrets are incorrect (randomly selected)", func(t *testing.T) {
			skrs := environment.getSkrsForSimulateChange([]int{})
			environment.simulateSecretChangeOnSkr(skrs)
			environment.assertAllSecretDataAreSet()
			stats, err := environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)
			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)
			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, len(skrs), stats.updatedCnt)
			assert.Equal(t, expectedInstancesCount-len(skrs), stats.updateErrorsCnt+stats.notChangedCnt)
			assert.Zero(t, stats.skippedCnt)

			environment.assertAllSecretDataAreSet()
			environment.assureConsistency()
		})

		t.Run("reconcile when some secrets are incorrect (static selected)", func(t *testing.T) {
			max := max(testDataIndexes)
			assert.GreaterOrEqual(t, expectedInstancesCount-1, max)
			skrs := environment.getSkrsForSimulateChange(testDataIndexes)
			environment.simulateSecretChangeOnSkr(skrs)
			environment.assertAllSecretDataAreSet()
			stats, err := environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)
			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)
			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, len(testDataIndexes), stats.updatedCnt)
			assert.Equal(t, expectedInstancesCount-len(testDataIndexes), stats.updateErrorsCnt+stats.notChangedCnt)
			assert.Zero(t, stats.skippedCnt)

			environment.assertAllSecretDataAreSet()
			environment.assureConsistency()
		})

		t.Run("when one secret is labeled, changed and relabelled", func(t *testing.T) {
			labeledSkrIdx := 0
			skrToBeSkipped := environment.getSkrsForSimulateChange([]int{labeledSkrIdx})
			labeledSkrInstanceId := environment.instanceIds[labeledSkrIdx]

			// when secret is labeled with skip-reconciliation
			environment.labelSecret(skrToBeSkipped[0].Config, skipReconciliationLabel, "true")

			maxIndex := max(testDataIndexes)
			assert.GreaterOrEqual(t, expectedInstancesCount-1, maxIndex)
			skrs := environment.getSkrsForSimulateChange(testDataIndexes)
			environment.simulateSecretChangeOnSkr(skrs)
			environment.assertAllSecretDataAreSet()

			// when we reconcile
			stats, err := environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)

			// then
			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)

			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, len(testDataIndexes)-1, stats.updatedCnt)
			assert.Equal(t, expectedInstancesCount-len(testDataIndexes), stats.updateErrorsCnt+stats.notChangedCnt)
			assert.Equal(t, 1, stats.skippedCnt)

			environment.assertAllSecretDataAreSet()
			environment.assureConsistencyExceptSkippedInstance(labeledSkrInstanceId)

			// when secret is updated by the user
			environment.setClusterID(skrToBeSkipped[0].Config, "custom-cluster-id")

			// when we reconcile again
			stats, err = environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)

			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)

			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, 0, stats.updatedCnt)
			assert.Equal(t, 1, stats.skippedCnt)
			environment.assertClusterID(skrToBeSkipped[0].Config, "custom-cluster-id")

			// then we remove the secret
			environment.removeSecretFromSkr(skrToBeSkipped[0].Config)

			// when we reconcile
			stats, err = environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)

			// then
			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)

			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, 1, stats.updatedCnt)
			assert.Zero(t, stats.skippedCnt)

			environment.assureConsistency()
			environment.assertAllSecretDataAreSet()

		})

		t.Run("when one secret is labeled, changed then removed", func(t *testing.T) {

			labeledSkrIdx := 0
			skrToBeSkipped := environment.getSkrsForSimulateChange([]int{labeledSkrIdx})
			labeledSkrInstanceId := environment.instanceIds[labeledSkrIdx]

			// when secret is labeled with skip-reconciliation
			environment.labelSecret(skrToBeSkipped[0].Config, skipReconciliationLabel, "true")

			maxIndex := max(testDataIndexes)
			assert.GreaterOrEqual(t, expectedInstancesCount-1, maxIndex)
			skrs := environment.getSkrsForSimulateChange(testDataIndexes)
			environment.simulateSecretChangeOnSkr(skrs)
			environment.assertAllSecretDataAreSet()

			// when we reconcile
			stats, err := environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)

			// then
			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)

			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, len(testDataIndexes)-1, stats.updatedCnt)
			assert.Equal(t, expectedInstancesCount-len(testDataIndexes), stats.updateErrorsCnt+stats.notChangedCnt)
			assert.Equal(t, 1, stats.skippedCnt)

			environment.assertAllSecretDataAreSet()
			environment.assureConsistencyExceptSkippedInstance(labeledSkrInstanceId)

			// when secret is updated by the user
			environment.setClusterID(skrToBeSkipped[0].Config, "custom-cluster-id")

			// when we reconcile again
			stats, err = environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)

			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)

			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, 0, stats.updatedCnt)
			assert.Equal(t, 1, stats.skippedCnt)
			environment.assertClusterID(skrToBeSkipped[0].Config, "custom-cluster-id")

			// then we change the label
			environment.labelSecret(skrToBeSkipped[0].Config, skipReconciliationLabel, "false")
			// when we reconcile

			stats, err = environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)

			// then
			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)

			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, 1, stats.updatedCnt)
			assert.Zero(t, stats.skippedCnt)

			environment.assureConsistency()
			environment.assertAllSecretDataAreSet()

		})

		t.Run("reconcile when one secret is labeled with skip-reconciliation set not to true", func(t *testing.T) {
			labeledSkrIdx := 0
			skrToBeSkipped := environment.getSkrsForSimulateChange([]int{labeledSkrIdx})
			environment.labelSecret(skrToBeSkipped[0].Config, skipReconciliationLabel, "not-true")

			maxIndex := max(testDataIndexes)
			assert.GreaterOrEqual(t, expectedInstancesCount-1, maxIndex)
			skrs := environment.getSkrsForSimulateChange(testDataIndexes)
			environment.simulateSecretChangeOnSkr(skrs)
			environment.assertAllSecretDataAreSet()

			stats, err := environment.manager.ReconcileAll(jobReconciliationDelay, nil)
			assert.NoError(t, err)

			environment.assertNumberOfInstancesInDb(expectedAllInstancesCount)

			assert.Equal(t, expectedInstancesCount, stats.instanceCnt)
			assert.Equal(t, len(testDataIndexes), stats.updatedCnt)
			assert.Equal(t, expectedInstancesCount-len(testDataIndexes), stats.updateErrorsCnt+stats.notChangedCnt)
			assert.Equal(t, 0, stats.skippedCnt)

			environment.assertAllSecretDataAreSet()
			environment.assureConsistency()
		})
		t.Cleanup(func() {
			cancel()
		})
	})
}

func TestManager(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	manager := Manager{
		logger: logger,
	}
	t.Run("compare secrets with all different data", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "b",
			ClientSecret:      "b",
			ServiceManagerURL: "b",
			URL:               "b",
			XSAppName:         "b",
		}, "b")
		assert.NoError(t, err)

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.NoError(t, err)
		assert.NotNil(t, notMatchingKeys)
		assert.Greater(t, len(notMatchingKeys), 0)
		assert.Equal(t, notMatchingKeys, []string{secretClientSecret, secretClientId, secretSmUrl, secretTokenUrl, secretClusterId})
	})

	t.Run("compare secrets with partially different data", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "b",
			ClientSecret:      "b",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.NoError(t, err)
		assert.NotNil(t, notMatchingKeys)
		assert.Greater(t, len(notMatchingKeys), 0)
		assert.Equal(t, notMatchingKeys, []string{secretClientSecret, secretClientId})
	})

	t.Run("compare secrets with the same data", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a1",
			ClientSecret:      "a2",
			ServiceManagerURL: "a3",
			URL:               "a4",
			XSAppName:         "a5",
		}, "a6")
		assert.NoError(t, err)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a1",
			ClientSecret:      "a2",
			ServiceManagerURL: "a3",
			URL:               "a4",
			XSAppName:         "a5",
		}, "a6")
		assert.NoError(t, err)

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.NoError(t, err)
		assert.NotNil(t, notMatchingKeys)
		assert.Equal(t, len(notMatchingKeys), 0)
	})

	t.Run("compare secrets where some of data is missing and data is same", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)
		delete(current.Data, secretClientSecret)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.Nil(t, notMatchingKeys)
		assert.Error(t, err)
	})

	t.Run("compare secrets where some of data is missing and data are different", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)
		delete(current.Data, secretClientSecret)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "b",
			ClientSecret:      "b",
			ServiceManagerURL: "b",
			URL:               "b",
			XSAppName:         "b",
		}, "b")
		assert.NoError(t, err)

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.Nil(t, notMatchingKeys)
		assert.Error(t, err)
	})
}

func (e *Environment) createTestData() {
	e.createClusters(expectedInstancesCount)
	e.instanceIds = make([]string, expectedInstancesCount)
	for i := 0; i < expectedInstancesCount; i++ {
		cfg := *e.skrs[i].Config
		clusterId := cfg.Host
		kubeConfig := restConfigToString(cfg)
		require.NotEmpty(e.t, kubeConfig)
		instanceId, runtimeId := e.createInstance(kubeConfig, generateServiceManagerCredentials(), clusterId)
		e.instanceIds[i] = instanceId
		e.createKyma(runtimeId, instanceId)
	}

	for i := 0; i < expectedRejectedInstancesCount; i++ {
		e.createInstance("", generateServiceManagerCredentials(), "")
	}
}

func (e *Environment) createClusters(skrCount int) {
	tempSkrs := make([]*envtest.Environment, skrCount)
	wg := &sync.WaitGroup{}

	// Create KCP
	wg.Add(1)
	go func() {
		defer wg.Done()
		testEnv := &envtest.Environment{
			CRDDirectoryPaths: []string{"testdata/crds/kyma.yaml"},
		}
		cfg, err := testEnv.Start()
		if err != nil {
			e.logs.Error(fmt.Sprintf("%v", err))
			return
		}
		k8sClient, err := client.New(cfg, client.Options{})
		if err != nil {
			e.logs.Error(fmt.Sprintf("%v", err))
			return
		}
		e.kcp = k8sClient

		namespace := &apicorev1.Namespace{}
		namespace.ObjectMeta = metav1.ObjectMeta{Name: kcpNamespace}
		err = e.kcp.Create(context.Background(), namespace)
		if err != nil {
			e.logs.Error(fmt.Sprintf("while creating KCP cluster: %v", err))
			return
		}
	}()

	// Create SKR Clusters
	for i := 0; i < skrCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			testEnv := &envtest.Environment{}
			_, err := testEnv.Start()
			if err != nil {
				e.logs.Error(fmt.Sprintf("while creating SKR cluster %v", err))
				return
			}

			tempSkrs[i] = testEnv
		}(i)
	}

	wg.Wait()
	e.skrs = append(e.skrs, tempSkrs...)
	require.Equal(e.t, len(e.skrs), skrCount)
	for _, skr := range e.skrs {
		require.NotNil(e.t, skr)
		require.NotEmpty(e.t, skr)
	}
	require.NotZero(e.t, e.skrs)
	require.NotNil(e.t, e.kcp)
}

func (e *Environment) createInstance(kubeConfig string, credentials *internal.ServiceManagerOperatorCredentials, clusterId string) (string, string) {
	instanceId, err := uuid2.NewUUID()
	require.NoError(e.t, err)

	runtimeId := ""
	reconcilable := false
	if kubeConfig != "" && clusterId != "" {
		runtimeUUID, err := uuid2.NewUUID()
		require.NoError(e.t, err)
		runtimeId = runtimeUUID.String()
		e.createKubeConfigSecret(kubeConfig, runtimeId)
		reconcilable = true
	}

	instance := &internal.Instance{
		InstanceID: instanceId.String(),
		RuntimeID:  runtimeId,
		InstanceDetails: internal.InstanceDetails{
			ServiceManagerClusterID: clusterId,
		},
		Parameters: internal.ProvisioningParameters{
			ErsContext: internal.ERSContext{
				SMOperatorCredentials: credentials,
			},
			Parameters: pkg.ProvisioningParametersDTO{
				Kubeconfig: kubeConfig,
			},
		},
	}
	instance.Reconcilable = reconcilable

	err = e.brokerStorage.Instances().Insert(*instance)
	require.NoError(e.t, err)
	return instanceId.String(), runtimeId
}

func (e *Environment) createKyma(runtimeId, instanceId string) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(KymaGvk)
	u.SetNamespace(kcpNamespace)
	u.SetName(runtimeId)
	labels := make(map[string]string, 1)
	labels[instanceIdLabel] = instanceId
	u.SetLabels(labels)
	err := e.kcp.Create(e.ctx, u)
	require.NoError(e.t, err)
}

func (e *Environment) createKubeConfigSecret(cfg, runtimeId string) {
	secret := &apicorev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getKubeConfigSecretName(runtimeId),
			Namespace: kcpNamespace,
		},
		Data: map[string][]byte{
			"config": []byte(cfg),
		},
		Type: apicorev1.SecretTypeOpaque,
	}
	err := e.kcp.Create(e.ctx, secret)
	require.NoError(e.t, err)
}

func (e *Environment) changeSecret(restCfg *rest.Config) {
	skrSecret := e.getSecretFromSkr(restCfg)
	newCredentials := generateServiceManagerCredentials()
	skrSecret.Data[secretClientSecret] = []byte(newCredentials.ClientSecret)
	skrSecret.Data[secretSmUrl] = []byte(newCredentials.ServiceManagerURL)
	skrSecret.Data[secretTokenUrl] = []byte(newCredentials.URL)
	skrSecret.Data[secretClusterId] = []byte(generateRandomText(credentialsLen))
	skrSecret.Data[secretClientId] = []byte(newCredentials.ClientID)
	e.updateSecretToSkr(restCfg, skrSecret)
}

func (e *Environment) setClusterID(restCfg *rest.Config, value string) {
	skrSecret := e.getSecretFromSkr(restCfg)
	require.NotNil(e.t, skrSecret)
	skrSecret.Data[secretClusterId] = []byte(value)
	e.updateSecretToSkr(restCfg, skrSecret)
}

func (e *Environment) assertClusterID(restCfg *rest.Config, value string) {
	skrSecret := e.getSecretFromSkr(restCfg)
	require.NotNil(e.t, skrSecret)
	assert.Equal(e.t, value, string(skrSecret.Data[secretClusterId]))
	e.updateSecretToSkr(restCfg, skrSecret)
}

func (e *Environment) labelSecret(restCfg *rest.Config, key, value string) {
	skrSecret := e.getSecretFromSkr(restCfg)
	require.NotNil(e.t, skrSecret)
	skrSecret.Labels = make(map[string]string, 1)
	skrSecret.Labels[key] = value
	e.updateSecretToSkr(restCfg, skrSecret)
}

func (e *Environment) removeSecretFromSkr(restCfg *rest.Config) {
	skrClient, err := client.New(restCfg, client.Options{})
	require.NoError(e.t, err)
	secret := &apicorev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: BtpManagerSecretNamespace,
			Name:      BtpManagerSecretName,
		}}

	err = skrClient.Delete(context.Background(), secret)
	require.NoError(e.t, err)
}

func (e *Environment) getSecretFromSkr(restCfg *rest.Config) *apicorev1.Secret {
	skrClient, err := client.New(restCfg, client.Options{})
	require.NoError(e.t, err)
	skrSecret := &apicorev1.Secret{}
	err = skrClient.Get(context.Background(), client.ObjectKey{Name: BtpManagerSecretName, Namespace: BtpManagerSecretNamespace}, skrSecret)
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	require.NoError(e.t, err)
	return skrSecret
}

func (e *Environment) updateSecretToSkr(restCfg *rest.Config, secret *apicorev1.Secret) {
	skrClient, err := client.New(restCfg, client.Options{})
	require.NoError(e.t, err)
	err = skrClient.Update(context.Background(), secret)
	require.NoError(e.t, err)
}

func (e *Environment) getSkrsForSimulateChange(skrIndexes []int) []*envtest.Environment {
	var result []*envtest.Environment
	if skrIndexes == nil || len(skrIndexes) == 0 {
		indexSet := map[int]struct{}{}
		for len(indexSet) != changedInstancesCount {

			random := rand.Intn(expectedInstancesCount)
			_, ok := indexSet[random]
			if !ok {
				indexSet[random] = struct{}{}
			}
		}

		for index := range indexSet {
			testEnv := e.skrs[index]
			result = append(result, testEnv)
		}
	} else {
		for _, index := range skrIndexes {
			testEnv := e.skrs[index]
			result = append(result, testEnv)
		}
	}
	return result
}

func (e *Environment) simulateSecretChangeOnSkr(skrs []*envtest.Environment) {
	for _, skr := range skrs {
		e.changeSecret(skr.Config)
	}
}

func (e *Environment) assertAllSecretsNotExists() {
	for _, skr := range e.skrs {
		skrSecret := e.getSecretFromSkr(skr.Config)
		require.Nil(e.t, skrSecret)
	}
}

func (e *Environment) assertAllSecretsExists() {
	for _, skr := range e.skrs {
		skrSecret := e.getSecretFromSkr(skr.Config)
		require.NotNil(e.t, skrSecret)
	}
}

func (e *Environment) assertAllSecretDataAreSet() {
	for _, skr := range e.skrs {
		skrSecret := e.getSecretFromSkr(skr.Config)
		require.NotNil(e.t, skrSecret)

		require.NotEmpty(e.t, getString(skrSecret.Data, secretClientId))
		require.NotEmpty(e.t, getString(skrSecret.Data, secretClientSecret))
		require.NotEmpty(e.t, getString(skrSecret.Data, secretSmUrl))
		require.NotEmpty(e.t, getString(skrSecret.Data, secretTokenUrl))
		require.NotEmpty(e.t, getString(skrSecret.Data, secretClusterId))

	}
}

func (e *Environment) assureConsistency() {
	instances, err := e.manager.GetReconcileCandidates()
	require.NoError(e.t, err)
	require.Equal(e.t, expectedInstancesCount, len(instances))

	for _, instance := range instances {
		skrK8sCfg, credentials := []byte(instance.Parameters.Parameters.Kubeconfig), instance.Parameters.ErsContext.SMOperatorCredentials
		restCfg, err := clientcmd.RESTConfigFromKubeConfig(skrK8sCfg)
		require.NoError(e.t, err)
		skrSecret := e.getSecretFromSkr(restCfg)
		require.NotNil(e.t, skrSecret)

		require.Equal(e.t, getString(skrSecret.Data, secretClientId), credentials.ClientID)
		require.Equal(e.t, getString(skrSecret.Data, secretClientSecret), credentials.ClientSecret)
		require.Equal(e.t, getString(skrSecret.Data, secretSmUrl), credentials.ServiceManagerURL)
		require.Equal(e.t, getString(skrSecret.Data, secretTokenUrl), credentials.URL)
		require.Equal(e.t, getString(skrSecret.Data, secretClusterId), instance.InstanceDetails.ServiceManagerClusterID)
	}
}

// TODO extend to make it more flexible
func (e *Environment) assureConsistencyExceptSkippedInstance(instanceID string) {
	instances, err := e.manager.GetReconcileCandidates()
	require.NoError(e.t, err)
	require.Equal(e.t, expectedInstancesCount, len(instances))

	for _, instance := range instances {
		if instance.InstanceID != instanceID {
			skrK8sCfg, credentials := []byte(instance.Parameters.Parameters.Kubeconfig), instance.Parameters.ErsContext.SMOperatorCredentials
			restCfg, err := clientcmd.RESTConfigFromKubeConfig(skrK8sCfg)
			require.NoError(e.t, err)
			skrSecret := e.getSecretFromSkr(restCfg)
			require.NotNil(e.t, skrSecret)

			require.Equal(e.t, getString(skrSecret.Data, secretClientId), credentials.ClientID)
			require.Equal(e.t, getString(skrSecret.Data, secretClientSecret), credentials.ClientSecret)
			require.Equal(e.t, getString(skrSecret.Data, secretSmUrl), credentials.ServiceManagerURL)
			require.Equal(e.t, getString(skrSecret.Data, secretTokenUrl), credentials.URL)
			require.Equal(e.t, getString(skrSecret.Data, secretClusterId), instance.InstanceDetails.ServiceManagerClusterID)
		}
	}
}

func (e *Environment) assureThatClusterIsInIncorrectState() int {
	instances, err := e.manager.GetReconcileCandidates()
	require.NoError(e.t, err)
	require.Equal(e.t, expectedInstancesCount, len(instances))

	incorrectClusters := 0
	for _, instance := range instances {
		require.NoError(e.t, err)
		skrK8sCfg, credentials := []byte(instance.Parameters.Parameters.Kubeconfig), instance.Parameters.ErsContext.SMOperatorCredentials
		restCfg, err := clientcmd.RESTConfigFromKubeConfig(skrK8sCfg)
		require.NoError(e.t, err)
		skrSecret := e.getSecretFromSkr(restCfg)
		require.NotNil(e.t, skrSecret)

		if getString(skrSecret.Data, secretClientSecret) != credentials.ClientSecret {
			incorrectClusters++
			continue
		}
		if getString(skrSecret.Data, secretClientId) != credentials.ClientID {
			incorrectClusters++
			continue
		}
		if getString(skrSecret.Data, secretTokenUrl) != credentials.URL {
			incorrectClusters++
			continue
		}
		if getString(skrSecret.Data, secretClusterId) != instance.InstanceDetails.ServiceManagerClusterID {
			incorrectClusters++
			continue
		}
	}

	return incorrectClusters
}

func (e *Environment) assertNumberOfInstancesInDb(expectedInstancesInDbCount int) {
	instances, _, _, err := e.brokerStorage.Instances().List(dbmodel.InstanceFilter{})
	require.NoError(e.t, err)
	require.Equal(e.t, expectedInstancesInDbCount, len(instances))
}

func restConfigToString(restConfig rest.Config) string {
	bytes, err := clientcmd.Write(api.Config{
		Clusters: map[string]*api.Cluster{
			"default": {
				Server:                   restConfig.Host,
				InsecureSkipTLSVerify:    restConfig.Insecure,
				CertificateAuthorityData: restConfig.CAData,
			},
		},
		Contexts: map[string]*api.Context{
			"default": {
				Cluster:  "default",
				AuthInfo: "default",
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"default": {
				ClientCertificateData: restConfig.CertData,
				ClientKeyData:         restConfig.KeyData,
				Token:                 restConfig.BearerToken,
				Username:              restConfig.Username,
				Password:              restConfig.Password,
			},
		},
		CurrentContext: "default",
	})
	if err != nil {
		return ""
	} else {
		return string(bytes)
	}
}

func generateServiceManagerCredentials() *internal.ServiceManagerOperatorCredentials {
	return &internal.ServiceManagerOperatorCredentials{
		ClientID:          generateRandomText(credentialsLen),
		ClientSecret:      generateRandomText(credentialsLen),
		ServiceManagerURL: generateRandomText(credentialsLen),
		URL:               generateRandomText(credentialsLen),
		XSAppName:         generateRandomText(credentialsLen),
	}
}

func generateRandomText(count int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	runes := make([]rune, count)
	for i := range runes {
		runes[i] = letterRunes[random.Intn(len(letterRunes))]
	}
	return string(runes)
}

func max(slice []int) int {
	max := 0
	for _, v := range slice {
		if v > max {
			max = v
		}
	}
	return max
}

func getString(m map[string][]byte, key string) string {
	value, ok := m[key]
	if !ok {
		return ""
	}
	return string(value)
}
