package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/kyma-project/kyma-environment-broker/internal"
	brokerBindings "github.com/kyma-project/kyma-environment-broker/internal/broker/bindings"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	expirationSeconds    = 600
	maxExpirationSeconds = 7200
	minExpirationSeconds = 600
)

func TestCreateBinding(t *testing.T) {
	// given

	// // schema
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
		}...).
		Build()

	// // secret check in assertions
	err = clientFirst.Create(context.Background(), &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "secret-to-check-first",
			Namespace: "default",
		},
	})
	require.NoError(t, err)

	bindingCfg := BindingConfig{
		Enabled: true,
		BindablePlans: EnablePlans{
			fixture.PlanName,
		},
		ExpirationSeconds:    expirationSeconds,
		MaxExpirationSeconds: maxExpirationSeconds,
		MinExpirationSeconds: minExpirationSeconds,
		MaxBindingsCount:     maxBindingsCount,
	}
	db := storage.NewMemoryStorage()

	err = db.Instances().Insert(fixture.FixInstance(instanceID1))
	require.NoError(t, err)
	operation := fixture.FixOperation("operation-001", instanceID1, internal.OperationTypeProvision)
	err = db.Operations().InsertOperation(operation)
	require.NoError(t, err)

	skrK8sClientProvider := kubeconfig.NewK8sClientFromSecretProvider(kcpClient)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	publisher := event.NewPubSub(log)
	svc := NewBind(bindingCfg, db, log, skrK8sClientProvider, skrK8sClientProvider, publisher)
	unbindSvc := NewUnbind(log, db, brokerBindings.NewServiceAccountBindingsManager(skrK8sClientProvider, skrK8sClientProvider), publisher)

	t.Run("should create a new service binding without error", func(t *testing.T) {
		// When
		response, err := svc.Bind(context.Background(), instanceID1, "binding-id", domain.BindDetails{
			AppGUID:       "",
			PlanID:        "",
			ServiceID:     "",
			BindResource:  nil,
			RawContext:    json.RawMessage(`{}`),
			RawParameters: json.RawMessage(`{"expiration_seconds": 660}`),
		}, false)

		// then
		require.NoError(t, err)

		assertClusterAccess(t, "secret-to-check-first", response.Credentials)
		assertRolesExistence(t, brokerBindings.BindingName("binding-id"), response.Credentials)
		assertTokenDuration(t, response.Credentials, 11*time.Minute)
	})

	t.Run("should create two bindings and unbind one of them", func(t *testing.T) {
		// when
		response1, err1 := svc.Bind(context.Background(), instanceID1, "binding-id1", domain.BindDetails{
			AppGUID:       "",
			PlanID:        "",
			ServiceID:     "",
			BindResource:  nil,
			RawContext:    json.RawMessage(`{}`),
			RawParameters: json.RawMessage(`{"expirationSeconds": 660}`),
		}, false)
		response2, err2 := svc.Bind(context.Background(), instanceID1, "binding-id2", domain.BindDetails{
			AppGUID:       "",
			PlanID:        "",
			ServiceID:     "",
			BindResource:  nil,
			RawContext:    json.RawMessage(`{}`),
			RawParameters: json.RawMessage(`{"expirationSeconds": 600}`),
		}, false)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)

		assertClusterAccess(t, "secret-to-check-first", response1.Credentials)
		assertClusterAccess(t, "secret-to-check-first", response2.Credentials)

		// when unbinding occurs for first binding, second should still work
		_, err := unbindSvc.Unbind(context.Background(), instanceID1, "binding-id1", domain.UnbindDetails{}, false)
		require.NoError(t, err)

		// then
		assertClusterNoAccess(t, "secret-to-check-first", response1.Credentials)
		assertClusterAccess(t, "secret-to-check-first", response2.Credentials)
	})
}

type kubeconfigStruct struct {
	Users []kubeconfigUser `yaml:"users"`
}

type kubeconfigUser struct {
	Name string `yaml:"name"`
	User struct {
		Token string `yaml:"token"`
	} `yaml:"user"`
}

func assertTokenDuration(t *testing.T, creds interface{}, expectedDuration time.Duration) {

	credentials, ok := creds.(Credentials)
	require.True(t, ok)
	config := credentials.Kubeconfig
	var kubeconfigObj kubeconfigStruct

	err := yaml.Unmarshal([]byte(config), &kubeconfigObj)
	require.NoError(t, err)

	var tokenDuration time.Duration
	for _, user := range kubeconfigObj.Users {
		if user.Name == "cluster-test" {
			token, _, err := new(jwt.Parser).ParseUnverified(user.User.Token, jwt.MapClaims{})
			require.NoError(t, err)

			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				iat := int64(claims["iat"].(float64))
				exp := int64(claims["exp"].(float64))

				issuedAt := time.Unix(iat, 0)
				expiresAt := time.Unix(exp, 0)

				tokenDuration = expiresAt.Sub(issuedAt)
				fmt.Printf("%v %v\n", expiresAt, issuedAt)
			} else {
				assert.Fail(t, "invalid token claims")
				break
			}
		}
	}

	assert.Equal(t, expectedDuration, tokenDuration)
}

func createEnvTest(t *testing.T) (envtest.Environment, *rest.Config, client.Client) {

	fmt.Println("setup envtest")

	pid := internal.SetupEnvtest(t)
	defer func() {
		internal.CleanupEnvtestBinaries(pid)
	}()

	fmt.Println("start envtest")

	env := envtest.Environment{
		ControlPlaneStartTimeout: 40 * time.Second,
	}
	var errEnvTest error
	var config *rest.Config
	err := wait.PollUntilContextTimeout(context.Background(), 500*time.Millisecond, 5*time.Second, true, func(context.Context) (done bool, err error) {
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

func initClient(cfg *rest.Config) (client.Client, error) {
	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, fmt.Errorf("while creating HTTP client for REST mapper: %w", err)
	}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, httpClient)
	if err != nil {
		err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
			mapper, err = apiutil.NewDynamicRESTMapper(cfg, httpClient)
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

func assertClusterAccess(t *testing.T, controlSecretName string, creds interface{}) {
	credentials, ok := creds.(Credentials)
	require.True(t, ok)
	kubeconfig := credentials.Kubeconfig

	newClient := kubeconfigClient(t, kubeconfig)

	_, err := newClient.CoreV1().Secrets("default").Get(context.Background(), controlSecretName, v1.GetOptions{})
	assert.NoError(t, err)
}

func assertClusterNoAccess(t *testing.T, controlSecretName string, creds interface{}) {
	credentials, ok := creds.(Credentials)
	require.True(t, ok)
	kubeconfig := credentials.Kubeconfig

	newClient := kubeconfigClient(t, kubeconfig)

	_, err := newClient.CoreV1().Secrets("default").Get(context.Background(), controlSecretName, v1.GetOptions{})
	assert.True(t, apierrors.IsForbidden(err))
}

func kubeconfigClient(t *testing.T, kubeconfig string) *kubernetes.Clientset {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	assert.NoError(t, err)

	clientset, err := kubernetes.NewForConfig(config)
	assert.NoError(t, err)

	return clientset
}

func assertRolesExistence(t *testing.T, bindingID string, creds interface{}) {
	credentials, ok := creds.(Credentials)
	require.True(t, ok)
	kubeconfig := credentials.Kubeconfig

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
