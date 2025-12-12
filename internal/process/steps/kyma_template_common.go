package steps

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8syaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	k8syamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

func DecodeKymaTemplate(kymaTemplate string) (*unstructured.Unstructured, error) {
	tmpl := []byte(kymaTemplate)

	decoder := k8syamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(tmpl), 512)
	var rawObj runtime.RawExtension
	if err := decoder.Decode(&rawObj); err != nil {
		return nil, err
	}
	obj, _, err := k8syaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return nil, err
	}

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
	return unstructuredObj, err
}

func EncodeKymaTemplate(kymaTemplate *unstructured.Unstructured) (string, error) {
	result, err := yaml.Marshal(kymaTemplate.Object)
	if err != nil {
		return "", fmt.Errorf("while marshal unstructured to yaml: %v", err)
	}
	return string(result), nil
}
