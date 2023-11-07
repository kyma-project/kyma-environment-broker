package internal

import (
	"embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// KEB tests can run in parallel resulting in concurrent access to scheme maps
// if the global scheme from client-go is used. For this reason, KEB tests each have
// their own scheme.
func NewSchemeForTests() *runtime.Scheme {
	sch := runtime.NewScheme()
	corev1.AddToScheme(sch)
	apiextensionsv1.AddToScheme(sch)
	return sch
}

//go:embed testdata/kymatemplate
var content embed.FS

func GetKymaTemplateForTests(t *testing.T, path string) string {
	file, err := content.ReadFile(fmt.Sprintf("%s/%s/%s", "testdata", "kymatemplate", path))
	assert.NoError(t, err)
	return string(file)
}
