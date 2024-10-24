package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	brokerBindings "github.com/kyma-project/kyma-environment-broker/internal/broker/bindings"
	"gopkg.in/yaml.v2"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/handlers"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

type Kubeconfig struct {
	Users []User `yaml:"users"`
}

type User struct {
	Name string `yaml:"name"`
	User struct {
		Token string `yaml:"token"`
	} `yaml:"user"`
}

const (
	instanceID1          = "1"
	instanceID2          = "2"
	instanceID3          = "max-bindings"
	expirationSeconds    = 10000
	maxExpirationSeconds = 7200
	minExpirationSeconds = 600
	bindingsPath         = "v2/service_instances/%s/service_bindings/%s"
	deleteParams         = "?accepts_incomplete=false&service_id=%s&plan_id=%s"
	maxBindingsCount     = 10
)

var httpServer *httptest.Server

func TestCreateBindingEndpoint(t *testing.T) {
	t.Log("test create binding endpoint")

	// Given
	//// logger
	logs := logrus.New()
	logs.SetLevel(logrus.DebugLevel)
	logs.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	brokerLogger := lager.NewLogger("test")
	brokerLogger.RegisterSink(lager.NewWriterSink(logs.Writer(), lager.DEBUG))

	//// schema
	sch := runtime.NewScheme()
	err := corev1.AddToScheme(sch)
	assert.NoError(t, err)

	// prepare envtest to provide valid kubeconfig
	envFirst, configFirst, clientFirst := createEnvTest(t)

	defer func(env *envtest.Environment) {
		err := env.Stop()
		assert.NoError(t, err)
	}(&envFirst)
	kbcfgFirst := createKubeconfigFileForRestConfig(*configFirst)

	//// secret check in assertions
	err = clientFirst.Create(context.Background(), &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "secret-to-check-first",
			Namespace: "default",
		},
	})
	require.NoError(t, err)

	// prepare envtest to provide valid kubeconfig for the second environment
	envSecond, configSecond, clientSecond := createEnvTest(t)

	defer func(env *envtest.Environment) {
		err := env.Stop()
		assert.NoError(t, err)
	}(&envSecond)
	kbcfgSecond := createKubeconfigFileForRestConfig(*configSecond)

	//// secret check in assertions
	err = clientSecond.Create(context.Background(), &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "secret-to-check-second",
			Namespace: "default",
		},
	})

	//// create fake kubernetes client - kcp
	kcpClient := fake.NewClientBuilder().
		WithScheme(sch).
		WithRuntimeObjects([]runtime.Object{
			&corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "kubeconfig-runtime-1",
					Namespace: "kcp-system",
				},
				Data: map[string][]byte{
					"config": kbcfgFirst,
				},
			},
			&corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "kubeconfig-runtime-2",
					Namespace: "kcp-system",
				},
				Data: map[string][]byte{
					"config": kbcfgSecond,
				},
			},
			&corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "kubeconfig-runtime-max-bindings",
					Namespace: "kcp-system",
				},
				Data: map[string][]byte{
					"config": kbcfgSecond,
				},
			},
		}...).
		Build()

	//// database
	storageCleanup, db, err := GetStorageForE2ETests()
	t.Cleanup(func() {
		if storageCleanup != nil {
			err := storageCleanup()
			assert.NoError(t, err)
		}
	})
	assert.NoError(t, err)

	err = db.Instances().Insert(fixture.FixInstance(instanceID1))
	require.NoError(t, err)
	operation := fixture.FixOperation("operation-001", instanceID1, internal.OperationTypeProvision)
	err = db.Operations().InsertOperation(operation)
	require.NoError(t, err)

	err = db.Instances().Insert(fixture.FixInstance(instanceID2))
	require.NoError(t, err)
	operation = fixture.FixOperation("operation-002", instanceID2, internal.OperationTypeProvision)
	err = db.Operations().InsertOperation(operation)
	require.NoError(t, err)

	err = db.Instances().Insert(fixture.FixInstance(instanceID3))
	require.NoError(t, err)
	operation = fixture.FixOperation("operation-003", instanceID3, internal.OperationTypeProvision)
	err = db.Operations().InsertOperation(operation)
	require.NoError(t, err)

	skrK8sClientProvider := kubeconfig.NewK8sClientFromSecretProvider(kcpClient)

	//// binding configuration
	bindingCfg := &broker.BindingConfig{
		Enabled: true,
		BindablePlans: broker.EnablePlans{
			fixture.PlanName,
		},
		ExpirationSeconds:    expirationSeconds,
		MaxExpirationSeconds: maxExpirationSeconds,
		MinExpirationSeconds: minExpirationSeconds,
		MaxBindingsCount:     maxBindingsCount,
	}

	//// api handler
	bindEndpoint := broker.NewBind(*bindingCfg, db, logs, skrK8sClientProvider, skrK8sClientProvider)
	getBindingEndpoint := broker.NewGetBinding(logs, db)
	unbindEndpoint := broker.NewUnbind(logs, db, brokerBindings.NewServiceAccountBindingsManager(skrK8sClientProvider, skrK8sClientProvider))
	apiHandler := handlers.NewApiHandler(broker.KymaEnvironmentBroker{
		ServicesEndpoint:             nil,
		ProvisionEndpoint:            nil,
		DeprovisionEndpoint:          nil,
		UpdateEndpoint:               nil,
		GetInstanceEndpoint:          nil,
		LastOperationEndpoint:        nil,
		BindEndpoint:                 bindEndpoint,
		UnbindEndpoint:               unbindEndpoint,
		GetBindingEndpoint:           getBindingEndpoint,
		LastBindingOperationEndpoint: nil,
	}, brokerLogger)

	//// attach bindings api
	router := mux.NewRouter()
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Bind).Methods(http.MethodPut)
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.GetBinding).Methods(http.MethodGet)
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Unbind).Methods(http.MethodDelete)
	httpServer = httptest.NewServer(router)
	defer httpServer.Close()

	t.Run("should create a new service binding without error", func(t *testing.T) {
		// When
		response := createBinding(instanceID1, "binding-id", t)
		defer response.Body.Close()

		binding := unmarshal(t, response)
		require.Equal(t, http.StatusCreated, response.StatusCode)

		duration, err := getTokenDurationFromBinding(t, binding)
		require.NoError(t, err)
		assert.Equal(t, expirationSeconds*time.Second, duration)

		assert.NotEmpty(t, binding.Metadata.ExpiresAt)

		//// verify connectivity using kubeconfig from the generated binding
		assertClusterAccess(t, "secret-to-check-first", binding)
		assertRolesExistence(t, brokerBindings.BindingName("binding-id"), binding)
	})

	t.Run("should create a new service binding with custom token expiration time", func(t *testing.T) {
		const customExpirationSeconds = 900

		// When
		response := createBindingWithCustomExpiration(instanceID1, "binding-id2", customExpirationSeconds, t)

		defer response.Body.Close()

		binding := unmarshal(t, response)
		require.Equal(t, http.StatusCreated, response.StatusCode)

		duration, err := getTokenDurationFromBinding(t, binding)
		require.NoError(t, err)
		assert.Equal(t, customExpirationSeconds*time.Second, duration)
	})

	t.Run("should return error when expiration_seconds is greater than maxExpirationSeconds", func(t *testing.T) {
		const customExpirationSeconds = 7201

		// When
		response := createBindingWithCustomExpiration(instanceID1, "binding-id3", customExpirationSeconds, t)

		defer response.Body.Close()
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
	})

	t.Run("should return error when expiration_seconds is less than minExpirationSeconds", func(t *testing.T) {
		const customExpirationSeconds = 60

		// When
		response := createBindingWithCustomExpiration(instanceID1, "binding-id4", customExpirationSeconds, t)

		defer response.Body.Close()
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
	})

	t.Run("should return 404 for not existing binding", func(t *testing.T) {
		// when
		response := getBinding(instanceID1, uuid.New().String(), t)
		defer response.Body.Close()

		// then
		require.Equal(t, http.StatusNotFound, response.StatusCode)
	})

	t.Run("should return created kubeconfig", func(t *testing.T) {
		// given
		bindingID := uuid.New().String()

		// when
		response := createBinding(instanceID1, bindingID, t)
		defer response.Body.Close()

		require.Equal(t, http.StatusCreated, response.StatusCode)

		// then
		binding := getBindingUnmarshalled(instanceID1, bindingID, t)

		duration, err := getTokenDurationFromBinding(t, binding)
		require.NoError(t, err)
		assert.Equal(t, expirationSeconds*time.Second, duration)

		//// verify connectivity using kubeconfig from the generated binding
		assertClusterAccess(t, "secret-to-check-first", binding)
	})

	t.Run("should return created bindings when multiple bindings created", func(t *testing.T) {
		// given
		firstInstanceFirstBindingID, firstInstancefirstBinding := createBindingWithRandomBindingID(instanceID1, httpServer, t)
		firstInstanceFirstBindingDB, err := db.Bindings().Get(instanceID1, firstInstanceFirstBindingID)
		assert.NoError(t, err)

		secondInstanceBindingID, secondInstanceFirstBinding := createBindingWithRandomBindingID(instanceID2, httpServer, t)
		secondInstanceFirstBindingDB, err := db.Bindings().Get(instanceID2, secondInstanceBindingID)
		assert.NoError(t, err)

		firstInstanceSecondBindingID, firstInstanceSecondBinding := createBindingWithRandomBindingID(instanceID1, httpServer, t)
		firstInstanceSecondBindingDB, err := db.Bindings().Get(instanceID1, firstInstanceSecondBindingID)
		assert.NoError(t, err)

		// when - first binding to the first instance

		response := getBinding(instanceID1, firstInstanceFirstBindingID, t)
		defer response.Body.Close()

		// then
		assert.Equal(t, http.StatusOK, response.StatusCode)
		binding := unmarshal(t, response)
		assertKubeconfigsEqual(t, firstInstancefirstBinding, binding)
		assert.Equal(t, firstInstanceFirstBindingDB.Kubeconfig, binding.Credentials.(map[string]interface{})["kubeconfig"])
		assertClusterAccess(t, "secret-to-check-first", binding)

		// when - binding to the second instance
		response = getBinding(instanceID2, secondInstanceBindingID, t)
		defer response.Body.Close()

		// then
		assert.Equal(t, http.StatusOK, response.StatusCode)
		binding = unmarshal(t, response)
		assertKubeconfigsEqual(t, secondInstanceFirstBinding, binding)
		assert.Equal(t, secondInstanceFirstBindingDB.Kubeconfig, binding.Credentials.(map[string]interface{})["kubeconfig"])
		assertClusterAccess(t, "secret-to-check-second", binding)

		// when - second binding to the first instance
		response = getBinding(instanceID1, firstInstanceSecondBindingID, t)
		defer response.Body.Close()

		// then
		assert.Equal(t, http.StatusOK, response.StatusCode)
		binding = unmarshal(t, response)
		assertKubeconfigsEqual(t, firstInstanceSecondBinding, binding)
		assert.Equal(t, firstInstanceSecondBindingDB.Kubeconfig, binding.Credentials.(map[string]interface{})["kubeconfig"])
		assertClusterAccess(t, "secret-to-check-first", binding)
	})

	t.Run("should delete created binding", func(t *testing.T) {
		// given
		bindingID, binding := createBindingWithRandomBindingID(instanceID1, httpServer, t)
		bindingFromDB, err := db.Bindings().Get(instanceID1, bindingID)
		assert.NoError(t, err)
		assert.Equal(t, binding.Credentials.(map[string]interface{})["kubeconfig"], bindingFromDB.Kubeconfig)

		// when
		path := fmt.Sprintf(bindingsPath+deleteParams, instanceID1, bindingID, "123", fixture.PlanId)
		response := CallAPI(httpServer, http.MethodDelete, path, "", t)
		defer response.Body.Close()

		// then
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bindingFromDB, err = db.Bindings().Get(instanceID1, bindingID)
		assert.Error(t, err)
		assert.Nil(t, bindingFromDB)
	})

	t.Run("should delete created binding and fail after the second call", func(t *testing.T) {
		// given
		bindingID, binding := createBindingWithRandomBindingID(instanceID1, httpServer, t)
		bindingFromDB, err := db.Bindings().Get(instanceID1, bindingID)
		assert.NoError(t, err)
		assert.Equal(t, binding.Credentials.(map[string]interface{})["kubeconfig"], bindingFromDB.Kubeconfig)

		// when
		path := fmt.Sprintf(bindingsPath+deleteParams, instanceID1, bindingID, "123", fixture.PlanId)
		response := CallAPI(httpServer, http.MethodDelete, path, "", t)
		defer response.Body.Close()

		// then
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bindingFromDB, err = db.Bindings().Get(instanceID1, bindingID)
		assert.Error(t, err)
		assert.Nil(t, bindingFromDB)

		// when
		path = fmt.Sprintf(bindingsPath+deleteParams, instanceID1, bindingID, "123", fixture.PlanId)
		response = CallAPI(httpServer, http.MethodDelete, path, "", t)
		defer response.Body.Close()

		// then
		assert.Equal(t, http.StatusGone, response.StatusCode)
		bindingFromDB, err = db.Bindings().Get(instanceID1, bindingID)
		assert.Error(t, err)
		assert.Nil(t, bindingFromDB)
	})

	t.Run("should selectively delete created binding and its service account resources", func(t *testing.T) {
		// given
		instanceFirst := "1"

		//// first instance first binding
		createdBindingIDInstanceFirstFirst, createdBindingInstanceFirstFirst := createBindingWithRandomBindingID(instanceFirst, httpServer, t)

		assertExistsAndKubeconfigCreated(t, createdBindingInstanceFirstFirst, createdBindingIDInstanceFirstFirst, instanceFirst, httpServer, db)

		assertResourcesExistence(t, clientFirst, createdBindingIDInstanceFirstFirst)

		//// first instance second binding
		createdBindingIDInstanceFirstSecond, createdBindingInstanceFirstSecond := createBindingWithRandomBindingID(instanceFirst, httpServer, t)

		assertExistsAndKubeconfigCreated(t, createdBindingInstanceFirstSecond, createdBindingIDInstanceFirstSecond, instanceFirst, httpServer, db)

		assertResourcesExistence(t, clientFirst, createdBindingIDInstanceFirstSecond)

		//// second instance first binding
		instanceSecond := "2"
		createdBindingIDInstanceSecondFirst, createdBindingInstanceSecondFirst := createBindingWithRandomBindingID(instanceSecond, httpServer, t)

		assertExistsAndKubeconfigCreated(t, createdBindingInstanceSecondFirst, createdBindingIDInstanceSecondFirst, instanceSecond, httpServer, db)

		assertResourcesExistence(t, clientSecond, createdBindingIDInstanceSecondFirst)

		//// second instance second binding
		createdBindingIDInstanceSecondSecond, createdBindingInstanceSecondSecond := createBindingWithRandomBindingID(instanceSecond, httpServer, t)

		assertExistsAndKubeconfigCreated(t, createdBindingInstanceSecondSecond, createdBindingIDInstanceSecondSecond, instanceSecond, httpServer, db)

		assertResourcesExistence(t, clientSecond, createdBindingIDInstanceSecondSecond)

		// when
		path := fmt.Sprintf(bindingsPath+deleteParams, instanceFirst, createdBindingIDInstanceFirstFirst, "123", fixture.PlanId)

		response := CallAPI(httpServer, http.MethodDelete, path, "", t)
		defer response.Body.Close()

		// then
		assert.Equal(t, http.StatusOK, response.StatusCode)

		assertServiceAccountsNotExists(t, clientFirst, createdBindingIDInstanceFirstFirst)

		assertExistsAndKubeconfigCreated(t, createdBindingInstanceFirstSecond, createdBindingIDInstanceFirstSecond, instanceFirst, httpServer, db)

		assertResourcesExistence(t, clientFirst, createdBindingIDInstanceFirstSecond)

		assertExistsAndKubeconfigCreated(t, createdBindingInstanceSecondFirst, createdBindingIDInstanceSecondFirst, instanceSecond, httpServer, db)

		assertResourcesExistence(t, clientSecond, createdBindingIDInstanceSecondFirst)

		assertExistsAndKubeconfigCreated(t, createdBindingInstanceSecondSecond, createdBindingIDInstanceSecondSecond, instanceSecond, httpServer, db)

		assertResourcesExistence(t, clientSecond, createdBindingIDInstanceSecondSecond)

		removedBinding, err := db.Bindings().Get(instanceFirst, createdBindingIDInstanceFirstFirst)
		assert.Error(t, err)
		assert.Nil(t, removedBinding)

	})

	t.Run("should return error when attempting to add a new binding when the maximum number of non expired bindings has already been reached", func(t *testing.T) {
		// given - create max number of bindings
		var response *http.Response

		for i := 0; i < maxBindingsCount; i++ {
			response = createBinding(instanceID3, uuid.New().String(), t)
			defer response.Body.Close()
			require.Equal(t, http.StatusCreated, response.StatusCode)
		}

		// when - create one more binding
		response = createBinding(instanceID3, uuid.New().String(), t)
		defer response.Body.Close()

		//then
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
	})
}

func assertResourcesExistence(t *testing.T, k8sClient client.Client, bindingID string) {
	name := brokerBindings.BindingName(bindingID)

	serviceAccount := corev1.ServiceAccount{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: brokerBindings.BindingNamespace}, &serviceAccount)
	assert.NoError(t, err)
	assert.NotNil(t, serviceAccount)

	clusterRole := rbacv1.ClusterRole{}
	err = k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: brokerBindings.BindingNamespace}, &clusterRole)
	assert.NoError(t, err)
	assert.NotNil(t, clusterRole)

	clusterRoleBinding := rbacv1.ClusterRoleBinding{}
	err = k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: brokerBindings.BindingNamespace}, &clusterRoleBinding)
	assert.NoError(t, err)
	assert.NotNil(t, clusterRoleBinding)
}

func assertServiceAccountsNotExists(t *testing.T, k8sClient client.Client, bindingID string) {
	name := brokerBindings.BindingName(bindingID)

	serviceAccount := corev1.ServiceAccount{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: brokerBindings.BindingNamespace}, &serviceAccount)
	assert.True(t, apierrors.IsNotFound(err))

	clusterRole := rbacv1.ClusterRole{}
	err = k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: brokerBindings.BindingNamespace}, &clusterRole)
	assert.True(t, apierrors.IsNotFound(err))

	clusterRoleBinding := rbacv1.ClusterRoleBinding{}
	err = k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: brokerBindings.BindingNamespace}, &clusterRoleBinding)
	assert.True(t, apierrors.IsNotFound(err))
}

func assertExistsAndKubeconfigCreated(t *testing.T, actual domain.Binding, bindingID, instanceID string, httpServer *httptest.Server, db storage.BrokerStorage) {
	expected, err := db.Bindings().Get(instanceID, bindingID)
	require.NoError(t, err)
	require.Equal(t, actual.Credentials.(map[string]interface{})["kubeconfig"], expected.Kubeconfig)
}

func assertClusterAccess(t *testing.T, controlSecretName string, binding domain.Binding) {

	credentials, ok := binding.Credentials.(map[string]interface{})
	require.True(t, ok)
	kubeconfig := credentials["kubeconfig"].(string)

	newClient := kubeconfigClient(t, kubeconfig)

	_, err := newClient.CoreV1().Secrets("default").Get(context.Background(), controlSecretName, v1.GetOptions{})
	assert.NoError(t, err)
}

func assertRolesExistence(t *testing.T, bindingID string, binding domain.Binding) {

	credentials, ok := binding.Credentials.(map[string]interface{})
	require.True(t, ok)
	kubeconfig := credentials["kubeconfig"].(string)

	newClient := kubeconfigClient(t, kubeconfig)

	_, err := newClient.CoreV1().ServiceAccounts(brokerBindings.BindingNamespace).Get(context.Background(), bindingID, v1.GetOptions{})
	assert.NoError(t, err)
	_, err = newClient.RbacV1().ClusterRoles().Get(context.Background(), bindingID, v1.GetOptions{})
	assert.NoError(t, err)
	_, err = newClient.RbacV1().ClusterRoleBindings().Get(context.Background(), bindingID, v1.GetOptions{})
	assert.NoError(t, err)
	_, err = newClient.RbacV1().ClusterRoleBindings().Get(context.Background(), bindingID, v1.GetOptions{})
	assert.NoError(t, err)
}

func assertKubeconfigsEqual(t *testing.T, expected, actual domain.Binding) {
	assert.Equal(t, expected.Credentials.(map[string]interface{})["kubeconfig"], actual.Credentials.(map[string]interface{})["kubeconfig"])
}

func createBindingWithRandomBindingID(instanceID string, httpServer *httptest.Server, t *testing.T) (string, domain.Binding) {
	bindingID := uuid.New().String()

	response := createBinding(instanceID, bindingID, t)
	defer response.Body.Close()
	require.Equal(t, http.StatusCreated, response.StatusCode)

	createdBinding := unmarshal(t, response)

	return bindingID, createdBinding
}

func createKubeconfigFileForRestConfig(restConfig rest.Config) []byte {
	const (
		userName    = "user"
		clusterName = "cluster"
		contextName = "context"
	)

	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters[clusterName] = &clientcmdapi.Cluster{
		Server:                   restConfig.Host,
		CertificateAuthorityData: restConfig.CAData,
	}
	contexts := make(map[string]*clientcmdapi.Context)
	contexts[contextName] = &clientcmdapi.Context{
		Cluster:  clusterName,
		AuthInfo: userName,
	}
	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos[userName] = &clientcmdapi.AuthInfo{
		ClientCertificateData: restConfig.CertData,
		ClientKeyData:         restConfig.KeyData,
	}
	clientConfig := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: contextName,
		AuthInfos:      authinfos,
	}
	kubeconfig, _ := clientcmd.Write(clientConfig)
	return kubeconfig
}

func CallAPI(httpServer *httptest.Server, method string, path string, body string, t *testing.T) *http.Response {
	cli := httpServer.Client()
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", httpServer.URL, path), bytes.NewBuffer([]byte(body)))
	req.Header.Set("X-Broker-API-Version", "2.14")

	require.NoError(t, err)

	resp, err := cli.Do(req)
	require.NoError(t, err)
	return resp
}

func createBindingWithCustomExpiration(instanceID string, bindingID string, customExpirationSeconds int, t *testing.T) *http.Response {
	path := getPath(instanceID, bindingID)
	return CallAPI(httpServer, http.MethodPut,
		path, fmt.Sprintf(`
		{
			"service_id": "123",
			"plan_id": "%s",
			"parameters": {
				"expiration_seconds": %v
			}
		}`, fixture.PlanId, customExpirationSeconds), t)
}

func createBinding(instanceID string, bindingID string, t *testing.T) *http.Response {
	path := getPath(instanceID, bindingID)
	return CallAPI(httpServer, http.MethodPut,
		path, fmt.Sprintf(`
		{
			"service_id": "123",
			"plan_id": "%s",
			"context": {
				"email": "john.smith@email.com",
				"origin": "origin"
			}
		}`, fixture.PlanId), t)
}

func getBinding(instanceID string, bindingID string, t *testing.T) *http.Response {
	return CallAPI(httpServer, http.MethodGet, getPath(instanceID, bindingID), "", t)
}

func getBindingUnmarshalled(instanceID string, bindingID string, t *testing.T) domain.Binding {
	response := getBinding(instanceID, bindingID, t)
	require.Equal(t, http.StatusOK, response.StatusCode)

	return unmarshal(t, response)
}

func getPath(instanceID, bindingID string) string {
	return fmt.Sprintf(bindingsPath, instanceID, bindingID)
}

func kubeconfigClient(t *testing.T, kubeconfig string) *kubernetes.Clientset {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	assert.NoError(t, err)

	clientset, err := kubernetes.NewForConfig(config)
	assert.NoError(t, err)

	return clientset
}

func unmarshal(t *testing.T, response *http.Response) domain.Binding {
	content, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	t.Logf("response content is: %v", string(content))

	assert.Contains(t, string(content), "credentials")

	var binding domain.Binding
	err = json.Unmarshal(content, &binding)
	require.NoError(t, err)

	t.Logf("binding: %v", binding.Credentials)

	return binding
}

func getTokenDurationFromBinding(t *testing.T, binding domain.Binding) (time.Duration, error) {
	credentials, ok := binding.Credentials.(map[string]interface{})
	require.True(t, ok)
	kubeconfig := credentials["kubeconfig"].(string)

	return getTokenDuration(t, kubeconfig)
}

func getTokenDuration(t *testing.T, config string) (time.Duration, error) {
	var kubeconfig Kubeconfig

	err := yaml.Unmarshal([]byte(config), &kubeconfig)
	require.NoError(t, err)

	for _, user := range kubeconfig.Users {
		if user.Name == "context" {
			token, _, err := new(jwt.Parser).ParseUnverified(user.User.Token, jwt.MapClaims{})
			require.NoError(t, err)

			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				iat := int64(claims["iat"].(float64))
				exp := int64(claims["exp"].(float64))

				issuedAt := time.Unix(iat, 0)
				expiresAt := time.Unix(exp, 0)

				return expiresAt.Sub(issuedAt), nil
			} else {
				return 0, fmt.Errorf("invalid token claims")
			}
		}
	}
	return 0, fmt.Errorf("user with name 'context' not found")
}

func createEnvTest(t *testing.T) (envtest.Environment, *rest.Config, client.Client) {
	pid := internal.SetupEnvtest(t)
	defer func() {
		internal.CleanupEnvtestBinaries(pid)
	}()

	env := envtest.Environment{
		ControlPlaneStartTimeout: 40 * time.Second,
	}
	var errEnvTest error
	var config *rest.Config
	err := wait.Poll(500*time.Millisecond, 5*time.Second, func() (done bool, err error) {
		config, errEnvTest = env.Start()
		if err != nil {
			t.Logf("envtest could not start, retrying: %s", errEnvTest.Error())
			return false, nil
		}
		t.Logf("envtest started")
		return true, nil
	})
	require.NoError(t, err)
	require.NoError(t, errEnvTest)

	skrClient, err := initClient(config)
	require.NoError(t, err)

	err = skrClient.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: brokerBindings.BindingNamespace,
		},
	})
	require.NoError(t, err)
	return env, config, skrClient
}
