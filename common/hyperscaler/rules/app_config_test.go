package rules

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	_ "istio.io/client-go/pkg/clientset/versioned/fake"
)

// Test for checking if expected format of
func TestAppConfig(t *testing.T) {

	// given

	// kubernetes client
	sch := internal.NewSchemeForTests(t)
	require.NotNil(t, sch)

	// helm ch
	ch, err := loader.Load("../../../resources/keb")
	require.NoError(t, err)
	require.NotNil(t, ch)

	values, err := chartutil.ReadValuesFile("../../../resources/keb/values.yaml")
	require.NotNil(t, values)
	require.NoError(t, err)


	values["hap"] = map[string]interface{}{
		"rule": []string{
			"aws",
		},
	}

	values, err = chartutil.ToRenderValues(ch, values, chartutil.ReleaseOptions{}, &chartutil.Capabilities{})

	resources, err := engine.Render(ch, values)
	require.NoError(t, err)
	require.NotNil(t, resources)

	clientBuilder := fake.NewClientBuilder()

	for filename, res := range resources {
		if filename == "keb/templates/NOTES.txt" {
			continue
		}
		res = strings.Trim(res, "\n")
		res = strings.Trim(res, " ")
		if res == "" || res == "\n" || strings.Contains(res, "istio") {
			continue
		}
		fmt.Println("res: " + res)
		decoder := scheme.Codecs.UniversalDeserializer()
		runtimeObject, groupVersionKind, err := decoder.Decode([]byte(res), nil, nil)
		require.NoError(t, err)
		fmt.Printf("groupVersionKind: %v\n", groupVersionKind)
		fmt.Printf("runtimeObject: %v\n", runtimeObject)

		clientBuilder.WithRuntimeObjects(runtimeObject)
	}

	cli := clientBuilder.Build()
	require.NotNil(t, cli)

	t.Run("app config map should contain data with rules", func(t *testing.T) {
		// when
		appConfig := &v1.ConfigMap{}
		err = cli.Get(t.Context(), client.ObjectKey{
			Name: "kcp-kyma-environment-broker",
		}, appConfig)

		// then
		require.NoError(t, err)
		require.NotNil(t, appConfig)

		data, ok := appConfig.Data["hapRule.yaml"]
		require.True(t, ok)
		require.Equal(t, "rule:\n- aws", data)
	})

	t.Run("keb deployment should contain env variable with file path", func(t *testing.T) {
		// when
		deployment := &appsv1.Deployment{}
		err = cli.Get(t.Context(), client.ObjectKey{
			Name: "kcp-kyma-environment-broker",
		}, deployment)

		// then
		require.NoError(t, err)
		require.NotNil(t, deployment)

		envFound := false
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == "kyma-environment-broker" {
				for _, env := range container.Env {
					if env.Name == "APP_HAP_RULE_FILE_PATH" &&
						env.Value == "/config/hapRule.yaml" {
						envFound = true
							break
					}
				}
			}
		}

		require.True(t, envFound)
	})

}
