// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"strings"
	"testing"

	"k8s.io/client-go/kubernetes"

	kv1b "k8s.io/api/admission/v1beta1"
	fakek8s "k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	v8oclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/typed/verrazzano/v1beta1"

	vzv1b "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
)

// TestValidateBinding tests validation of VerrazzanoBinding
// GIVEN a VerrazzanoBinding, a VerrazzanoModel and a fake k8s client with existing secrets
//  WHEN validateBinding is called with the VerrazzanoBinding and the Clientsets
//  THEN the validation should produce a result message containing the expected substrings
func TestValidateBinding(t *testing.T) {
	model := ReadModel("testdata/bobs-books-v2-model.yaml")
	review := kv1b.AdmissionReview{Request: &kv1b.AdmissionRequest{Namespace: model.Namespace}}
	binding := ReadBinding("testdata/bobs-books-v2-binding.yaml")
	badBinding := ReadBinding("testdata/bobs-books-v2-binding-invalid.yaml")
	cluster := &vzv1b.VerrazzanoManagedCluster{}
	cluster.Namespace = model.Namespace
	cluster.Name = "local"
	sec := newSecret("default", "mysql-credentials", "hello")
	tests := []struct {
		name      string
		k8sClient kubernetes.Interface
		v8oClient v8oclientset.VerrazzanoV1beta1Interface
		binding   *vzv1b.VerrazzanoBinding
		//empty expectedErrorMessages array if the validation result is positive
		expectedErrorMessages []string
	}{
		{
			name:      "TestValidateBinding",
			k8sClient: fakek8s.NewSimpleClientset(sec),
			v8oClient: NewFakeVzClient(model, binding, cluster),
			binding:   binding,
		}, {
			name:                  "TestValidateBindingWithoutCluster",
			k8sClient:             fakek8s.NewSimpleClientset(sec),
			v8oClient:             NewFakeVzClient(model, binding),
			binding:               binding,
			expectedErrorMessages: []string{"binding references cluster(s) \"local\" that do not exist in namespace default"},
		}, {
			name:                  "TestValidateBindingMissingSecret",
			k8sClient:             fakek8s.NewSimpleClientset(),
			v8oClient:             NewFakeVzClient(model, binding, cluster),
			binding:               binding,
			expectedErrorMessages: []string{"binding references databaseBindings.credentials \"mysql-credentials\" for mysql"},
		}, {
			name:                  "TestValidateInvalidBinding",
			k8sClient:             fakek8s.NewSimpleClientset(sec),
			v8oClient:             NewFakeVzClient(model, badBinding, cluster),
			binding:               badBinding,
			expectedErrorMessages: []string{"Multiple occurrence of component across placement namespaces"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientsets := &Clientsets{V8oClient: test.v8oClient, K8sClient: test.k8sClient}
			admissionReview := validateBinding(review, *test.binding, clientsets, "myVerrazzanoURI")
			if len(test.expectedErrorMessages) == 0 {
				assert.Nil(t, admissionReview.Response)
			} else {
				errorMessage := admissionReview.Response.Result.Message
				for _, s := range test.expectedErrorMessages {
					if !strings.Contains(errorMessage, s) {
						t.Errorf("Error %v should contain %v", errorMessage, test.expectedErrorMessages)
					}
				}
			}
		})
	}
}
