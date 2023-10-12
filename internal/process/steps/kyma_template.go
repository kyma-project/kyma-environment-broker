package steps

import (
	"bytes"
	"fmt"
	cyaml2 "gopkg.in/yaml.v2"
	"log"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

type InitKymaTemplate struct {
	operationManager *process.OperationManager
}

var _ process.Step = &InitKymaTemplate{}

func NewInitKymaTemplate(os storage.Operations) *InitKymaTemplate {
	return &InitKymaTemplate{operationManager: process.NewOperationManager(os)}
}

func (s *InitKymaTemplate) Name() string {
	return "Init_Kyma_Template"
}

func (s *InitKymaTemplate) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	tmpl := operation.InputCreator.Configuration().KymaTemplate
	obj, err := DecodeKymaTemplate(tmpl)
	if err != nil {
		logger.Errorf("Unable to create kyma template: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	logger.Infof("Decoded kyma template: %v", obj)

	modules := operation.ProvisioningParameters.Parameters.Modules
	if err := appendModules(obj, modules); err != nil {
		logger.Errorf("Unable to append modules to kyma template: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "Unable to append modules to kyma template:", err, logger)
	}
	tmpl, err = encodeKymaTemplate(obj)
	if err != nil {
		logger.Errorf("Unable to create yaml kyma template : %s", err.Error())
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	fmt.Println("Yaml will be")
	fmt.Println(tmpl)
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = obj.GetNamespace()
		op.KymaTemplate = tmpl
	}, logger)
}

// NOTE: adapter for upgrade_kyma which is currently not using shared staged_manager
type initKymaTemplateUpgradeKyma struct {
	*InitKymaTemplate
}

func InitKymaTemplateUpgradeKyma(os storage.Operations) initKymaTemplateUpgradeKyma {
	return initKymaTemplateUpgradeKyma{NewInitKymaTemplate(os)}
}

func (s initKymaTemplateUpgradeKyma) Run(o internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation, w, err := s.InitKymaTemplate.Run(o.Operation, logger)
	return internal.UpgradeKymaOperation{operation}, w, err
}

func DecodeKymaTemplate(template string) (*unstructured.Unstructured, error) {
	tmpl := []byte(template)
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(tmpl), 512)
	var rawObj runtime.RawExtension
	if err := decoder.Decode(&rawObj); err != nil {
		return nil, err
	}
	obj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return nil, err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
	return unstructuredObj, err
}

func appendModules(out *unstructured.Unstructured, m *internal.ModulesDTO) error {
	//To consider using -> unstructured.SetNestedSlice()
	template := out.Object
	specSection := template["spec"]
	spec := specSection.(map[string]interface{})
	modulesSection := spec["modules"]
	if m == nil || m.List == nil || len(m.List) == 0 {
		return nil
	}

	toInsert := make([]interface{}, len(m.List))
	for i := range m.List {
		toInsert[i] = m.List[i]
	}
	modulesSection = toInsert
	spec["modules"] = modulesSection
	out.Object["spec"] = specSection

	unstructured.SetNestedField(out.Object, toInsert, "spec", "modules")

	return nil
}

func encodeKymaTemplate(tmpl *unstructured.Unstructured) (string, error) {
	out, err := cyaml2.Marshal(tmpl.Object)
	if err != nil {
		log.Fatal(err)
	}
	//mt.Printf("output:\n%s\n", out)
	return string(out), nil
}
