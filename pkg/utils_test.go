// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"

	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/rest"

	vzv1b "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v8oclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/typed/verrazzano/v1beta1"
	vzcli "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/typed/verrazzano/v1beta1"
	ktesting "k8s.io/client-go/testing"
)

// ReadModel reads/unmarshalls VerrazzanoModel yaml file into a VerrazzanoModel
func ReadModel(path string) *vzv1b.VerrazzanoModel {
	path = "../test/integ/" + path
	b, err := readYamlToJSON(path)
	if err != nil {
		zap.S().Errorf("Error reading model file %v", err)
	}
	var model vzv1b.VerrazzanoModel
	err = json.Unmarshal(b, &model)
	if err != nil {
		zap.S().Errorf("Error reading model file %v", err)
	}
	return &model
}

// ReadModel reads/unmarshalls VerrazzanoModel yaml file into a VerrazzanoModel
func ReadBinding(path string) *vzv1b.VerrazzanoBinding {
	path = "../test/integ/" + path
	b, err := readYamlToJSON(path)
	if err != nil {
		zap.S().Errorf("Error reading binding file %v", err)
	}
	var binding vzv1b.VerrazzanoBinding
	err = json.Unmarshal(b, &binding)
	if err != nil {
		zap.S().Errorf("Error reading binding file %v", err)
	}
	return &binding
}

func readYamlToJSON(path string) ([]byte, error) {
	filename, _ := filepath.Abs(path)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var body interface{}
	err = yaml.Unmarshal(yamlFile, &body)
	if err != nil {
		return nil, err
	}
	body = yaml2json(body)
	return json.Marshal(body)
}

func yaml2json(obj interface{}) interface{} {
	switch node := obj.(type) {
	case []interface{}:
		jsonArray := make([]interface{}, len(node))
		for index, item := range node {
			jsonArray[index] = yaml2json(item)
		}
		return jsonArray
	case map[interface{}]interface{}:
		jsonMap := map[string]interface{}{}
		for key, val := range node {
			jsonMap[key.(string)] = yaml2json(val)
		}
		return jsonMap
	default:
		return obj
	}
}

func newSecret(namespace, name, secret string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque, //corev1.SecretTypeOpaque
		StringData: map[string]string{
			"password": secret,
			"username": name,
		},
	}
}

// FakeVzClient implements VerrazzanoV1beta1Interface.
type FakeVzClient struct {
	ktesting.Fake
	//discovery *fakediscovery.FakeDiscovery
	tracker ktesting.ObjectTracker
}

func (f *FakeVzClient) RESTClient() rest.Interface {
	panic("implement this if needed")
}

var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory(scheme)

func init() {
	metav1.AddToGroupVersion(scheme, vzv1b.SchemeGroupVersion)
	utilruntime.Must(vzv1b.AddToScheme(scheme))
}

// NewFakeVzClient returns a clientset that will respond with the provided objects.
func NewFakeVzClient(objects ...runtime.Object) *FakeVzClient {
	o := ktesting.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}
	cs := &FakeVzClient{tracker: o}
	cs.AddReactor("*", "*", ktesting.ObjectReaction(o))
	return cs
}

//MockError mocks a fake kubernetes.Interface to return an expected error
func MockError(kubeCli *FakeVzClient, verb, resource string, obj runtime.Object) *FakeVzClient {
	kubeCli.PrependReactor(verb, resource,
		func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, obj, fmt.Errorf("Error %s %s", verb, resource)
		})
	return kubeCli
}

type FakeVzModels struct {
	Fake *FakeVzClient
	ns   string
}

func (f *FakeVzClient) VerrazzanoModels(namespace string) vzcli.VerrazzanoModelInterface {
	return &FakeVzModels{f, namespace}
}

func (f *FakeVzClient) VerrazzanoBindings(namespace string) vzcli.VerrazzanoBindingInterface {
	return &FakeVzBindings{f, namespace}
}

func (f *FakeVzClient) VerrazzanoManagedClusters(namespace string) vzcli.VerrazzanoManagedClusterInterface {
	return &FakeVzManagedClusters{f, namespace}
}

// verrazzano-crd-generator/deploy/crds/verrazzano.io_verrazzanomodels_crd.yaml
var vzModelResource = schema.GroupVersionResource{Group: "verrazzano.io", Version: "v1beta1", Resource: "verrazzanomodels"}
var vzModelKind = schema.GroupVersionKind{Group: "verrazzano.io", Version: "v1beta1", Kind: "VerrazzanoModel"}

// vendor/k8s.io/client-go/kubernetes/typed/apps/v1/fake/fake_deployment.go
func (f FakeVzModels) Create(ctx context.Context, verrazzanoModel *vzv1b.VerrazzanoModel, opts metav1.CreateOptions) (*vzv1b.VerrazzanoModel, error) {
	obj, err := f.Fake.
		Invokes(ktesting.NewCreateAction(vzModelResource, f.ns, verrazzanoModel), &vzv1b.VerrazzanoModel{})

	if obj == nil {
		return nil, err
	}
	return obj.(*vzv1b.VerrazzanoModel), err
}

func (f FakeVzModels) Update(ctx context.Context, verrazzanoModel *vzv1b.VerrazzanoModel, opts metav1.UpdateOptions) (*vzv1b.VerrazzanoModel, error) {
	obj, err := f.Fake.
		Invokes(ktesting.NewUpdateAction(vzModelResource, f.ns, verrazzanoModel), &vzv1b.VerrazzanoModel{})

	if obj == nil {
		return nil, err
	}
	return obj.(*vzv1b.VerrazzanoModel), err
}

func (f FakeVzModels) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	panic("implement this if needed")
}

func (f FakeVzModels) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("implement this if needed")
}

func (f FakeVzModels) Get(ctx context.Context, name string, opts metav1.GetOptions) (*vzv1b.VerrazzanoModel, error) {
	obj, err := f.Fake.
		Invokes(ktesting.NewGetAction(vzModelResource, f.ns, name), &vzv1b.VerrazzanoModel{})
	if obj == nil {
		return nil, err
	}
	return obj.(*vzv1b.VerrazzanoModel), err
}

func (f FakeVzModels) List(ctx context.Context, opts metav1.ListOptions) (*vzv1b.VerrazzanoModelList, error) {
	obj, err := f.Fake.
		Invokes(ktesting.NewListAction(vzModelResource, vzModelKind, f.ns, opts), &vzv1b.VerrazzanoModelList{})
	if obj == nil {
		return nil, err
	}
	label, _, _ := ktesting.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &vzv1b.VerrazzanoModelList{ListMeta: obj.(*vzv1b.VerrazzanoModelList).ListMeta}
	for _, item := range obj.(*vzv1b.VerrazzanoModelList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

func (f FakeVzModels) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("implement this if needed")
}

func (f FakeVzModels) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *vzv1b.VerrazzanoModel, err error) {
	panic("implement this if needed")
}

type FakeVzBindings struct {
	Fake *FakeVzClient
	ns   string
}

type FakeVzManagedClusters struct {
	Fake *FakeVzClient
	ns   string
}

// verrazzano-crd-generator/deploy/crds/verrazzano.io_verrazzanobindings_crd.yaml
var vzManagedClusterResource = schema.GroupVersionResource{Group: "verrazzano.io", Version: "v1beta1", Resource: "verrazzanomanagedclusters"}
var vzManagedClusterKind = schema.GroupVersionKind{Group: "verrazzano.io", Version: "v1beta1", Kind: "VerrazzanoManagedCluster"}

func (f FakeVzManagedClusters) Create(ctx context.Context, verrazzanoManagedCluster *vzv1b.VerrazzanoManagedCluster, opts metav1.CreateOptions) (*vzv1b.VerrazzanoManagedCluster, error) {
	panic("implement this if needed")
}

func (f FakeVzManagedClusters) Update(ctx context.Context, verrazzanoManagedCluster *vzv1b.VerrazzanoManagedCluster, opts metav1.UpdateOptions) (*vzv1b.VerrazzanoManagedCluster, error) {
	panic("implement this if needed")
}

func (f FakeVzManagedClusters) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	panic("implement this if needed")
}

func (f FakeVzManagedClusters) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("implement this if needed")
}

func (f FakeVzManagedClusters) Get(ctx context.Context, name string, opts metav1.GetOptions) (*vzv1b.VerrazzanoManagedCluster, error) {
	obj, err := f.Fake.
		Invokes(ktesting.NewGetAction(vzManagedClusterResource, f.ns, name), &vzv1b.VerrazzanoManagedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*vzv1b.VerrazzanoManagedCluster), err
}

func (f FakeVzManagedClusters) List(ctx context.Context, opts metav1.ListOptions) (*vzv1b.VerrazzanoManagedClusterList, error) {
	panic("implement this if needed")
}

func (f FakeVzManagedClusters) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("implement this if needed")
}

func (f FakeVzManagedClusters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *vzv1b.VerrazzanoManagedCluster, err error) {
	panic("implement this if needed")
}

func configMapOf(name string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{}
	configMap.Namespace = "default"
	configMap.Name = name
	return configMap
}

func TestGetModel(t *testing.T) {
	model := ReadModel("testdata/bobs-books-v2-model.yaml")
	binding := &vzv1b.VerrazzanoBinding{}
	binding.Namespace = model.Namespace
	binding.Name = model.Name + "Binding"
	tests := []struct {
		name        string
		vzClient    v8oclientset.VerrazzanoV1beta1Interface
		expectedErr bool
	}{
		{
			name:        "TestGetModel",
			vzClient:    NewFakeVzClient(model, binding),
			expectedErr: false,
		},
		{
			name:        "TestGetModelWithErrors",
			vzClient:    MockError(NewFakeVzClient(model, binding), "get", "verrazzanomodels", &vzv1b.VerrazzanoModel{}),
			expectedErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			get, err := test.vzClient.VerrazzanoModels(model.Namespace).Get(context.TODO(), model.Name, metav1.GetOptions{})
			if (err != nil) != test.expectedErr {
				t.Errorf("VerrazzanoModels.Get() error = %v, wantErr %v", err, test.expectedErr)
			} else {
				assert.NotNil(t, get, "Expected get VerrazzanoModel")
			}
		})
	}
}

func TestListModel(t *testing.T) {
	model := ReadModel("testdata/bobs-books-v2-model.yaml")
	binding := &vzv1b.VerrazzanoBinding{}
	binding.Namespace = model.Namespace
	binding.Name = model.Name + "Binding"
	vzcli := NewFakeVzClient(model, binding)
	list, err := vzcli.VerrazzanoModels(model.Namespace).List(context.TODO(), metav1.ListOptions{})
	assert.Nil(t, err)
	assert.True(t, len(list.Items) > 0, "Expected get VerrazzanoModel")
}

func TestListModelWithError(t *testing.T) {
	model := ReadModel("testdata/bobs-books-v2-model.yaml")
	binding := &vzv1b.VerrazzanoBinding{}
	binding.Namespace = model.Namespace
	binding.Name = model.Name + "Binding"
	vzcli := NewFakeVzClient(model, binding)
	vzcli = MockError(vzcli, "list", "verrazzanomodels", &vzv1b.VerrazzanoModelList{})
	_, err := vzcli.VerrazzanoModels(model.Namespace).List(context.TODO(), metav1.ListOptions{})
	assert.NotNil(t, err, "Expected List error")
}

// verrazzano-crd-generator/deploy/crds/verrazzano.io_verrazzanobindings_crd.yaml
var vzBindingResource = schema.GroupVersionResource{Group: "verrazzano.io", Version: "v1beta1", Resource: "verrazzanobindings"}
var vzBindingKind = schema.GroupVersionKind{Group: "verrazzano.io", Version: "v1beta1", Kind: "VerrazzanoBinding"}

func (f FakeVzBindings) Create(ctx context.Context, verrazzanoBinding *vzv1b.VerrazzanoBinding, opts metav1.CreateOptions) (*vzv1b.VerrazzanoBinding, error) {
	panic("implement this if needed")
}

func (f FakeVzBindings) Update(ctx context.Context, verrazzanoBinding *vzv1b.VerrazzanoBinding, opts metav1.UpdateOptions) (*vzv1b.VerrazzanoBinding, error) {
	panic("implement this if needed")
}

func (f FakeVzBindings) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	panic("implement this if needed")
}

func (f FakeVzBindings) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("implement this if needed")
}

func (f FakeVzBindings) Get(ctx context.Context, name string, opts metav1.GetOptions) (*vzv1b.VerrazzanoBinding, error) {
	obj, err := f.Fake.
		Invokes(ktesting.NewGetAction(vzBindingResource, f.ns, name), &vzv1b.VerrazzanoBinding{})
	if obj == nil {
		return nil, err
	}
	return obj.(*vzv1b.VerrazzanoBinding), err
}

func (f FakeVzBindings) List(ctx context.Context, opts metav1.ListOptions) (*vzv1b.VerrazzanoBindingList, error) {
	obj, err := f.Fake.
		Invokes(ktesting.NewListAction(vzBindingResource, vzBindingKind, f.ns, opts), &vzv1b.VerrazzanoBindingList{})
	if obj == nil {
		return nil, err
	}
	label, _, _ := ktesting.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &vzv1b.VerrazzanoBindingList{ListMeta: obj.(*vzv1b.VerrazzanoBindingList).ListMeta}
	for _, item := range obj.(*vzv1b.VerrazzanoBindingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

func (f FakeVzBindings) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("implement this if needed")
}

func (f FakeVzBindings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *vzv1b.VerrazzanoBinding, err error) {
	panic("implement this if needed")
}

func TestGetBinding(t *testing.T) {
	model := ReadModel("testdata/bobs-books-v2-model.yaml")
	binding := &vzv1b.VerrazzanoBinding{}
	binding.Namespace = model.Namespace
	binding.Name = model.Name + "Binding"
	tests := []struct {
		name          string
		vzClient      v8oclientset.VerrazzanoV1beta1Interface
		expectedError bool
	}{
		{
			name:          "TestGetBinding",
			vzClient:      NewFakeVzClient(model, binding),
			expectedError: false,
		},
		{
			name:          "TestGetBindingWithErrors",
			vzClient:      MockError(NewFakeVzClient(model, binding), "get", "verrazzanobindings", &vzv1b.VerrazzanoBinding{}),
			expectedError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			get, err := test.vzClient.VerrazzanoBindings(binding.Namespace).Get(context.TODO(), binding.Name, metav1.GetOptions{})
			if (err != nil) != test.expectedError {
				t.Errorf("VerrazzanoBindings.Get() error = %v, wantErr %v", err, test.expectedError)
			} else {
				assert.NotNil(t, get, "Expected get VerrazzanoBinding")
			}
		})
	}
}

func TestListBinding(t *testing.T) {
	model := ReadModel("testdata/bobs-books-v2-model.yaml")
	binding := &vzv1b.VerrazzanoBinding{}
	binding.Namespace = model.Namespace
	binding.Name = model.Name + "Binding"
	vzClient := NewFakeVzClient(model, binding)
	list, err := vzClient.VerrazzanoBindings(binding.Namespace).List(context.TODO(), metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(list.Items), "Expected size of VerrazzanoBindings")
}

func TestListBindingError(t *testing.T) {
	model := ReadModel("testdata/bobs-books-v2-model.yaml")
	binding := &vzv1b.VerrazzanoBinding{}
	binding.Namespace = model.Namespace
	binding.Name = model.Name + "Binding"
	vzClient := NewFakeVzClient(model, binding)
	vzClient = MockError(vzClient, "list", "verrazzanobindings", &vzv1b.VerrazzanoBindingList{})
	_, err := vzClient.VerrazzanoBindings(binding.Namespace).List(context.TODO(), metav1.ListOptions{})
	assert.NotNil(t, err, "Expected List error")
}
