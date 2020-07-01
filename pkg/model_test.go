// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

// Testing validatePort
func TestValidatePort(t *testing.T) {
	// Test for ports out of bounds
	badPorts := []int{-1, 10000000, 0}
	for _, port := range badPorts {
		if err := validatePort(port); err == "" {
			t.Errorf("validate port failed with: %d", port)
		}
	}

	// Test for ports within range
	goodPorts := []int{1, 65535, 80}
	for _, port := range goodPorts {
		if err := validatePort(port); err != "" {
			t.Errorf("validate port failed with: %d", port)
		}
	}
}

// Test validateRestConnections
func TestValidateRestConnections(t *testing.T) {
	// Test for incorrect environment variable names
	restConnBad := []v1beta1v8o.VerrazzanoRestConnection{{"test", "TEST_HOST", "*TEST_PORT"},
		{"test2", "!TEST2_HOST", "TEST2_PORT"}}

	if err := validateRestConnections(restConnBad); err == "" {
		t.Error("Rest connections with invalid ports not recognized")
	}

	// Test for valid environment variable names
	restConnGood := []v1beta1v8o.VerrazzanoRestConnection{{"test", "TEST_HOST", "TEST_PORT"},
		{"test2", "TEST2_HOST", "TEST2_PORT"}}

	if err := validateRestConnections(restConnGood); err != "" {
		t.Error("Rest connections with valid ports were incorrectly invalidated")
	}
}

// Test validateWebLogicDomains
func TestValidateWebLogicDomains(t *testing.T) {
	// Test Valid Model
	vModel := v1beta1v8o.VerrazzanoModel{metav1.TypeMeta{Kind: "VerrizzanoModel", APIVersion: "1"},
		metav1.ObjectMeta{},
		v1beta1v8o.VerrazzanoModelSpec{},
		v1beta1v8o.VerrazzanoModelStatus{}}

	if err := validateWebLogicDomains(vModel); err != "" {
		t.Errorf("WebLogic domain failed with: %s", err)
	}

	// Test invalid model with AdminPort and T3Port out of bounds and equal to each other
	vModel2 := v1beta1v8o.VerrazzanoModel{metav1.TypeMeta{Kind: "VerrizzanoModel", APIVersion: "1"},
		metav1.ObjectMeta{},
		v1beta1v8o.VerrazzanoModelSpec{WeblogicDomains: []v1beta1v8o.VerrazzanoWebLogicDomain{{AdminPort: -1, T3Port: -1}}},
		v1beta1v8o.VerrazzanoModelStatus{}}
	if err := validateWebLogicDomains(vModel2); err == "" {
		t.Error("Invalid WebLogic domain falsely passed")
	}
}

// Test validateHelidonApplications
func TestValidateHelidonApplications(t *testing.T) {

	// Test valid model
	vModel := v1beta1v8o.VerrazzanoModel{metav1.TypeMeta{Kind: "VerrizzanoModel", APIVersion: "1"},
		metav1.ObjectMeta{},
		v1beta1v8o.VerrazzanoModelSpec{},
		v1beta1v8o.VerrazzanoModelStatus{}}

	if err := validateHelidonApplications(vModel); err != "" {
		t.Errorf("Helidon applications verification failed with: %s", err)
	}

	// Test model with Port and TargetPort out of bounds
	vModel2 := v1beta1v8o.VerrazzanoModel{metav1.TypeMeta{Kind: "VerrizzanoModel", APIVersion: "1"},
		metav1.ObjectMeta{},
		v1beta1v8o.VerrazzanoModelSpec{HelidonApplications: []v1beta1v8o.VerrazzanoHelidon{{Port: 0, TargetPort: 10000000}}},
		v1beta1v8o.VerrazzanoModelStatus{}}
	if err := validateHelidonApplications(vModel2); err == "" {
		t.Error("Invalid Helidon applications verification falsely passed")
	}
}

// Test validateCoherenceClusters
func TestValidateCoherenceClusters(t *testing.T) {

	// Test valid model
	vModel := v1beta1v8o.VerrazzanoModel{metav1.TypeMeta{Kind: "VerrizzanoModel", APIVersion: "1"},
		metav1.ObjectMeta{},
		v1beta1v8o.VerrazzanoModelSpec{},
		v1beta1v8o.VerrazzanoModelStatus{}}

	if err := validateCoherenceClusters(vModel); err != "" {
		t.Errorf("Coherence cluster validation failed with: %s", err)
	}

	// Test model with incorrect environment variable names
	vModel2 := v1beta1v8o.VerrazzanoModel{TypeMeta: metav1.TypeMeta{Kind: "VerrizzanoModel", APIVersion: "1"},
		Spec: v1beta1v8o.VerrazzanoModelSpec{CoherenceClusters: []v1beta1v8o.VerrazzanoCoherenceCluster{
			{Connections: []v1beta1v8o.VerrazzanoConnections{
				{Rest: []v1beta1v8o.VerrazzanoRestConnection{{"test", "@$%^TEST_HOST", "*TEST_PORT"},
					{"test2", "!TEST2_HOST", "TEST2_PORT"}}}}}}}}
	if err := validateCoherenceClusters(vModel2); err == "" {
		t.Error("Invalid Coherence cluster validation falsely passed")
	}
}