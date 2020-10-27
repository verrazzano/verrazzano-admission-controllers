// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes"

	vzv1b "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
)

// TestValidateGenericComponents tests validation of GenericComponents
// GIVEN a VerrazzanoModel and a fake k8s client
//  WHEN validateGenericComponents is called with the VerrazzanoModel and the Clientsets
//  THEN the validation should produce expected result message
func TestValidateGenericComponents(t *testing.T) {
	model := ReadModel("testdata/bobs-books-v2-model.yaml")
	binding := &vzv1b.VerrazzanoBinding{}
	binding.Namespace = model.Namespace
	binding.Name = model.Name + "Binding"
	type args struct {
		model      vzv1b.VerrazzanoModel
		kubeClient kubernetes.Interface
	}
	tests := []struct {
		name           string
		args           args
		expectedErrors []string
	}{
		{
			name: "testGenericComponents",
			args: args{model: *model, kubeClient: fakek8s.NewSimpleClientset(configMapOf("mysql-initdb-config"))},
		},
		{
			name: "testMissingConfigMap",
			args: args{model: *model, kubeClient: fakek8s.NewSimpleClientset()},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if errorMessage := validateGenericComponents(test.args.model); len(test.expectedErrors) > 0 {
				for _, s := range test.expectedErrors {
					if !strings.Contains(errorMessage, s) {
						t.Errorf("Error %v should contain %v", errorMessage, test.expectedErrors)
					}
				}
			}
		})
	}
}

// TestValidateModel tests validation of VerrazzanoModel
// GIVEN a VerrazzanoModel and a fake k8s client with existing secrets
//  WHEN validateModel is called with the VerrazzanoModel and the Clientsets
//  THEN the validation should produce a result message containing the expected substrings
func TestValidateModel(t *testing.T) {
	model := ReadModel("testdata/bobs-books-v2-model.yaml")
	binding := &vzv1b.VerrazzanoBinding{}
	binding.Namespace = model.Namespace
	binding.Name = model.Name + "Binding"
	secrets := []*corev1.Secret{
		newSecret("default", "ocr", "hello-ocr"),
		newSecret("default", "github-packages", "github-packages"),
		newSecret("default", "bobbys-front-end-weblogic-credentials", "hello"),
		newSecret("default", "bobs-bookstore-weblogic-credentials", "hello"),
		newSecret("default", "mysql-credentials", "hello")}

	model2 := ReadModel("testdata/bobs-books-v2-model.yaml")
	model2.Spec.GenericComponents[0].Deployment.InitContainers = model2.Spec.GenericComponents[0].Deployment.Containers

	tests := []struct {
		name                    string
		k8sClient               kubernetes.Interface
		model                   *vzv1b.VerrazzanoModel
		expectedErrorSubstrings []string
	}{
		{
			name:      "TestValidateModel",
			k8sClient: fakek8s.NewSimpleClientset(secrets[0], secrets[1], secrets[2], secrets[3], secrets[4]),
			model:     model,
		}, {
			name:                    "TestValidateModelWithMissingSecret",
			k8sClient:               fakek8s.NewSimpleClientset(),
			model:                   model,
			expectedErrorSubstrings: []string{"imagePullSecret", "default"},
		}, {
			name:                    "TestValidateModelWithMissingEnvSecret",
			k8sClient:               fakek8s.NewSimpleClientset(secrets[0], secrets[1], secrets[2], secrets[3]),
			model:                   model,
			expectedErrorSubstrings: []string{"mysql-credentials", "default"},
		}, {
			name:                    "TestValidateModelWithInitContainer",
			k8sClient:               fakek8s.NewSimpleClientset(secrets[0], secrets[1], secrets[2], secrets[3]),
			model:                   model2,
			expectedErrorSubstrings: []string{"mysql-credentials", "default"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientsets := &Clientsets{V8oClient: NewFakeVzClient(test.model, binding), K8sClient: test.k8sClient}
			admissionReview := validateModel(*test.model, clientsets)
			if len(test.expectedErrorSubstrings) == 0 {
				assert.Nil(t, admissionReview.Response)
			} else {
				for _, s := range test.expectedErrorSubstrings {
					errorMessage := admissionReview.Response.Result.Message
					if !strings.Contains(errorMessage, s) {
						t.Errorf("Error %v should contain %v", errorMessage, test.expectedErrorSubstrings)
					}
				}
			}
		})
	}
}
