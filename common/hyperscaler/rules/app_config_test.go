package rules

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

const HELM_CHART = "../../../resources/keb"
const BROKER_CONTAINER_NAME = "kyma-environment-broker"
const BROKER_CHART_NAME = "kcp-kyma-environment-broker"
const VALUES_FILE = HELM_CHART + "/values.yaml"
const NOTES_FILE = "keb/templates/NOTES.txt"

const ENV_NAME = "APP_HAP_RULE_FILE_PATH"
const ENV_FILENAME = "hapRule.yaml"
const ENV_PATH = "/config/" + ENV_FILENAME

func TestAppConfig(t *testing.T) {

	// given
	// kubernetes client
	sch := internal.NewSchemeForTests(t)
	require.NotNil(t, sch)

	// helm chart
	ch, err := loader.Load(HELM_CHART)
	require.NoError(t, err)
	require.NotNil(t, ch)

	values, err := chartutil.ReadValuesFile(VALUES_FILE)
	require.NotNil(t, values)
	require.NoError(t, err)

	values["hap"] = map[string]interface{}{
		"rule": []string{
			"aws",
		},
	}

	values, err = chartutil.ToRenderValues(ch, values, chartutil.ReleaseOptions{}, &chartutil.Capabilities{})
	require.NoError(t, err)

	resources, err := engine.Render(ch, values)
	require.NoError(t, err)
	require.NotNil(t, resources)

	clientBuilder := fake.NewClientBuilder()

	for filename, res := range resources {
		if filename == NOTES_FILE {
			continue
		}
		res = strings.Trim(res, "\n ")
		if res == "" || strings.Contains(res, "istio") {
			continue
		}

		decoder := scheme.Codecs.UniversalDeserializer()
		runtimeObject, _, err := decoder.Decode([]byte(res), nil, nil)
		require.NoError(t, err)

		clientBuilder.WithRuntimeObjects(runtimeObject)
	}

	cli := clientBuilder.Build()
	require.NotNil(t, cli)

	t.Run("app config map should contain data with rules", func(t *testing.T) {
		// when
		appConfig := &v1.ConfigMap{}
		err = cli.Get(context.Background(), client.ObjectKey{
			Name: BROKER_CHART_NAME,
		}, appConfig)

		// then
		require.NoError(t, err)
		require.NotNil(t, appConfig)

		data, ok := appConfig.Data[ENV_FILENAME]
		require.True(t, ok)
		require.Equal(t, "rule:\n- aws", data)
	})

	t.Run("keb deployment should contain env variable with file path", func(t *testing.T) {
		// when
		deployment := &appsv1.Deployment{}
		err = cli.Get(context.Background(), client.ObjectKey{
			Name: BROKER_CHART_NAME,
		}, deployment)

		// then
		require.NoError(t, err)
		require.NotNil(t, deployment)

		containers := deployment.Spec.Template.Spec.Containers
		cIndex := slices.IndexFunc(containers, func(c v1.Container) bool { return c.Name == BROKER_CONTAINER_NAME })

		found := slices.ContainsFunc(containers[cIndex].Env, func(e v1.EnvVar) bool { return e.Name == ENV_NAME && e.Value == ENV_PATH })

		require.True(t, found)
	})

}
