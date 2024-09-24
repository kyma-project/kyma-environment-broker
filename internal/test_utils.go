package internal

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

const (
	envTestAssets = "KUBEBUILDER_ASSETS"
)

//go:embed testdata/kymatemplate
var content embed.FS
var envtestInstallMutex sync.Mutex

// KEB tests can run in parallel resulting in concurrent access to scheme maps
// if the global scheme from client-go is used. For this reason, KEB tests each have
// their own scheme.
func NewSchemeForTests(t *testing.T) *k8sruntime.Scheme {
	sch := k8sruntime.NewScheme()
	err := corev1.AddToScheme(sch)
	assert.NoError(t, err)
	err = apiextensionsv1.AddToScheme(sch)
	assert.NoError(t, err)
	err = imv1.AddToScheme(sch)
	assert.NoError(t, err)
	return sch
}

func GetKymaTemplateForTests(t *testing.T, path string) string {
	file, err := content.ReadFile(fmt.Sprintf("%s/%s/%s", "testdata", "kymatemplate", path))
	assert.NoError(t, err)
	return string(file)
}

func SetupEnvtest(t *testing.T) int {
	_, currentPath, _, _ := runtime.Caller(0)
	script := fmt.Sprintf("%s/envtest.sh", path.Join(path.Dir(currentPath), "../"))
	envtestInstallMutex.Lock()
	defer envtestInstallMutex.Unlock()
	cmd := exec.Command("/bin/sh", script)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	fmt.Println(fmt.Sprintf("script process PID; %d", cmd.Process.Pid))
	fmt.Println(fmt.Sprintf("envtest setup output: %s err: %s \n", out.String(), stderr.String()))
	if err != nil {
		require.NoError(t, err)
	}
	require.NotEmpty(t, out)
	assets := out.String()
	assets = strings.Replace(assets, "\n", "", -1)
	err = os.Setenv(envTestAssets, assets)
	require.NoError(t, err)
	return cmd.Process.Pid
}

func CleanupEnvtestBinaries(pid int) {
	_, currentPath, _, _ := runtime.Caller(0)
	script := fmt.Sprintf("%s/clean-envtest-binaries.sh", path.Join(path.Dir(currentPath), "../"))
	cmd := exec.Command("/bin/sh", script, fmt.Sprintf("%d", pid))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	_ = cmd.Run()
	fmt.Println(fmt.Sprintf("envtest binaries cleanup output: %s err: %s \n", out.String(), stderr.String()))
}
