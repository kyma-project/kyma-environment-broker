package broker

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
	"gopkg.in/yaml.v2"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal"
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
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
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

const expirationSeconds = 10000
const maxExpirationSeconds = 7200
const minExpirationSeconds = 600

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
	pid := internal.SetupEnvtest(t)
	defer func() {
		internal.CleanupEnvtestBinaries(pid)
	}()

	env := envtest.Environment{
		ControlPlaneStartTimeout: 40 * time.Second,
	}
	var errEnvTest error
	var config *rest.Config
	err = wait.Poll(500*time.Millisecond, 5*time.Second, func() (done bool, err error) {
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
	defer func(env *envtest.Environment) {
		err := env.Stop()
		assert.NoError(t, err)
	}(&env)
	kbcfg := createKubeconfigFileForRestConfig(*config)
	skrClient, err := initClient(config)
	require.NoError(t, err)

	err = skrClient.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: "kyma-system",
		},
	})
	require.NoError(t, err)

	//// secret check in assertions
	err = skrClient.Create(context.Background(), &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "secret-to-check",
			Namespace: "default",
		},
	})

	require.NoError(t, err)

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
					"config": kbcfg,
				},
			},
		}...).
		Build()

	//// create fake kubernetes client - kcp
	gardenerClient := fake.NewClientBuilder().
		WithScheme(sch).
		WithRuntimeObjects([]runtime.Object{}...).
		Build()

	//// database
	db := storage.NewMemoryStorage()
	err = db.Instances().Insert(fixture.FixInstance("1"))
	require.NoError(t, err)

	skrK8sClientProvider := kubeconfig.NewK8sClientFromSecretProvider(kcpClient)

	//// binding configuration
	bindingCfg := &BindingConfig{
		Enabled: true,
		BindablePlans: EnablePlans{
			fixture.PlanName,
		},
		ExpirationSeconds:    expirationSeconds,
		MaxExpirationSeconds: maxExpirationSeconds,
		MinExpirationSeconds: minExpirationSeconds,
	}

	//// api handler
	bindEndpoint := NewBind(*bindingCfg, db.Instances(), logs, skrK8sClientProvider, skrK8sClientProvider, gardenerClient)
	apiHandler := handlers.NewApiHandler(KymaEnvironmentBroker{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		bindEndpoint,
		nil,
		nil,
		nil,
	}, brokerLogger)

	//// attach bindings api
	method := "PUT"
	router := mux.NewRouter()
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Bind).Methods(method)
	httpServer := httptest.NewServer(router)
	defer httpServer.Close()

	t.Run("should create a new service binding without error", func(t *testing.T) {

		// When
		response := CallAPI(httpServer, method, "v2/service_instances/1/service_bindings/binding-id?accepts_incomplete=true", fmt.Sprintf(`
		{
			"service_id": "123",
			"plan_id": "%s",
			"parameters": {
				"service_account": true
			}
		}`, fixture.PlanId), t)

		binding := verifyResponse(t, response)

		credentials, ok := binding.Credentials.(map[string]interface{})
		require.True(t, ok)
		kubeconfig := credentials["kubeconfig"].(string)

		duration, err := getTokenDuration(t, kubeconfig)
		require.NoError(t, err)
		assert.Equal(t, expirationSeconds*time.Second, duration)

		//// verify connectivity using kubeconfig from the generated binding
		newClient := kubeconfigClient(t, kubeconfig)
		_, err = newClient.CoreV1().Secrets("default").Get(context.Background(), "secret-to-check", v1.GetOptions{})
		assert.NoError(t, err)

		_, err = newClient.CoreV1().ServiceAccounts("kyma-system").Get(context.Background(), "kyma-binding-binding-id", v1.GetOptions{})
		assert.NoError(t, err)
		_, err = newClient.RbacV1().ClusterRoles().Get(context.Background(), "kyma-binding-binding-id", v1.GetOptions{})
		assert.NoError(t, err)
		_, err = newClient.RbacV1().ClusterRoleBindings().Get(context.Background(), "kyma-binding-binding-id", v1.GetOptions{})
		assert.NoError(t, err)
	})

	t.Run("should create a new service binding with custom token expiration time", func(t *testing.T) {
		const customExpirationSeconds = 900

		// When
		response := CallAPI(httpServer, method, "v2/service_instances/1/service_bindings/binding-id2?accepts_incomplete=true", fmt.Sprintf(`
		{
			"service_id": "123",
			"plan_id": "%s",
			"parameters": {
				"service_account": true,
				"expiration_seconds": %v
			}
		}`, fixture.PlanId, customExpirationSeconds), t)

		binding := verifyResponse(t, response)

		credentials, ok := binding.Credentials.(map[string]interface{})
		require.True(t, ok)

		duration, err := getTokenDuration(t, credentials["kubeconfig"].(string))
		require.NoError(t, err)
		assert.Equal(t, customExpirationSeconds*time.Second, duration)
	})
	t.Run("should return error when expiration_seconds is greater than maxExpirationSeconds", func(t *testing.T) {
		const customExpirationSeconds = 7201

		// When
		response := CallAPI(httpServer, method, "v2/service_instances/1/service_bindings/binding-id3?accepts_incomplete=true", fmt.Sprintf(`
		{
			"service_id": "123",
			"plan_id": "%s",
			"parameters": {
				"service_account": true,
				"expiration_seconds": %v

			}
		}`, fixture.PlanId, customExpirationSeconds), t)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
	})

	t.Run("should return error when expiration_seconds is less than minExpirationSeconds", func(t *testing.T) {
		const customExpirationSeconds = 60

		// When
		response := CallAPI(httpServer, method, "v2/service_instances/1/service_bindings/binding-id4?accepts_incomplete=true", fmt.Sprintf(`
		{
			"service_id": "123",
			"plan_id": "%s",
			"parameters": {	
				"service_account": true,
				"expiration_seconds": %v
			}	
		}`, fixture.PlanId, customExpirationSeconds), t)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
	})
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
	require.NoError(t, err)

	resp, err := cli.Do(req)
	require.NoError(t, err)
	return resp
}

func initClient(cfg *rest.Config) (client.Client, error) {
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
			mapper, err = apiutil.NewDiscoveryRESTMapper(cfg)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, fmt.Errorf("while waiting for client mapper: %w", err)
		}
	}
	cli, err := client.New(cfg, client.Options{Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("while creating a client: %w", err)
	}
	return cli, nil
}

func kubeconfigClient(t *testing.T, kubeconfig string) *kubernetes.Clientset {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	assert.NoError(t, err)

	clientset, err := kubernetes.NewForConfig(config)
	assert.NoError(t, err)

	return clientset
}

func verifyResponse(t *testing.T, response *http.Response) domain.Binding {
	require.Equal(t, http.StatusCreated, response.StatusCode)

	content, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	defer response.Body.Close()

	t.Logf("response content is: %v", string(content))

	assert.Contains(t, string(content), "credentials")

	var binding domain.Binding
	err = json.Unmarshal(content, &binding)
	require.NoError(t, err)

	t.Logf("binding: %v", binding.Credentials)

	return binding
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
