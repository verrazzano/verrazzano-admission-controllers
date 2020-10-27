// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"

	s "strings"

	"github.com/golang/glog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"k8s.io/api/admission/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sValidations "k8s.io/apimachinery/pkg/util/validation"
)

// Validate binding
func validateBinding(arRequest v1beta1.AdmissionReview, binding v1beta1v8o.VerrazzanoBinding, clientsets *Clientsets, verrazzanoURI string) v1beta1.AdmissionReview {
	// Don't allow create if the binding refers to a non-existing model
	modelList, err := clientsets.V8oClient.VerrazzanoModels(arRequest.Request.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil && modelList != nil {
		modelFound := false
		for _, model := range modelList.Items {
			if binding.Spec.ModelName == model.Name {
				modelFound = true
				break
			}
		}
		if !modelFound {
			message := fmt.Sprintf("binding is referencing model %s that does not exist in namespace %s", binding.Spec.ModelName, arRequest.Request.Namespace)
			glog.Error(message)
			return errorAdmissionReview(message)
		}
	}

	// All names that reference a k8s name must be valid.
	response := validateBindingResourceNames(binding)
	if response != "" {
		return errorAdmissionReview(response)
	}

	// Verify that the length of the VMI domain name is not greater than 64
	const VmiDomainNameFormat = "*.vmi.%s.%s"
	const MaxVmiDomainNameLen = 64
	domainName := fmt.Sprintf(VmiDomainNameFormat, binding.Name, verrazzanoURI)
	domainNameLen := len(domainName)
	if domainNameLen > MaxVmiDomainNameLen {
		message := fmt.Sprintf("the VMI domain name is greater than %d characters: %s.  The binding name %s is %d characters long.  Reduce the size by using a binding name that is at least %d characters shorter.", MaxVmiDomainNameLen, domainName, binding.Name, len(binding.Name), domainNameLen-MaxVmiDomainNameLen)
		glog.Error(message)
		return errorAdmissionReview(message)
	}

	// All placements names in the binding must have a matching VerrazzanoManagedClusters custom resource
	response = validateClusters(arRequest, binding, clientsets)
	if response != "" {
		return errorAdmissionReview(response)
	}

	response = validatePlacementNamespaces(binding)
	if response != "" {
		return errorAdmissionReview(response)
	}

	// Validate Ingress Bindings
	errMessages := validateIngressBinding(binding.Spec.IngressBindings)
	if len(errMessages) > 0 {
		return errorAdmissionReview(s.Join(errMessages, ", "))
	}

	// Validate components in the binding
	errMessages = validateComponents(arRequest, binding, clientsets)
	if len(errMessages) > 0 {
		return errorAdmissionReview(s.Join(errMessages, ", "))
	}

	// All secrets in the binding must be defined in the default namespace.
	response = validateBindingSecrets(binding, clientsets)
	if response != "" {
		return errorAdmissionReview(response)
	}

	glog.Info("validation of binding successful")
	return v1beta1.AdmissionReview{}
}

// Validate names that will be used as Kubernetes resource names.
// A validate k8s resource name must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an
// alphanumeric character.  We use k8s validation functions to check the validity of names.
func validateBindingResourceNames(binding v1beta1v8o.VerrazzanoBinding) string {
	glog.V(6).Info("In validateBindingResourceNames code")

	var errMessages []string

	// Check if namespace names are valid
	for i, placement := range binding.Spec.Placement {
		for j, namespace := range placement.Namespaces {
			field := fmt.Sprintf("spec.placement[%d].namespaces[%d].name", i, j)
			errMessages = addInvalidNameFormatMessage(namespace.Name, field, errMessages)
		}
	}

	// Check if database credentials names are valid
	for i, dbBinding := range binding.Spec.DatabaseBindings {
		field := fmt.Sprintf("spec.databaseBindings[%d].credential", i)
		errMessages = addInvalidNameFormatMessage(dbBinding.Credentials, field, errMessages)
	}

	if len(errMessages) > 0 {
		return s.Join(errMessages, "")
	}

	return ""
}

// Validate that the default namespace is not used in a binding placement
func validatePlacementNamespaces(binding v1beta1v8o.VerrazzanoBinding) string {
	glog.V(6).Info("In validatePlacementNamespaces code")

	for _, placement := range binding.Spec.Placement {
		for _, namespace := range placement.Namespaces {
			if namespace.Name == "default" {
				message := "default namespace is not allowed in placements of binding"
				glog.Error(message)
				return message
			}
		}
	}

	return ""
}

// Validate componets in the binding
func validateComponents(arRequest v1beta1.AdmissionReview, binding v1beta1v8o.VerrazzanoBinding, clientsets *Clientsets) []string {
	glog.V(6).Info("In validateComponents code")

	var errMessages []string
	// Get all components referenced in the binding
	componentsInBindingSet := make(map[string]bool)

	// All components should only occur once across all binding types being validated within the current binding yaml.
	for _, coherenceBinding := range binding.Spec.CoherenceBindings {
		if !componentsInBindingSet[coherenceBinding.Name] {
			componentsInBindingSet[coherenceBinding.Name] = true
		} else {
			errMessages = append(errMessages, fmt.Sprintf("Multiple occurrence of component for Coherence binding. Invalid Component: [%s]\n", coherenceBinding.Name))
		}
	}
	for _, helidonBinding := range binding.Spec.HelidonBindings {
		if !componentsInBindingSet[helidonBinding.Name] {
			componentsInBindingSet[helidonBinding.Name] = true
		} else {
			errMessages = append(errMessages, fmt.Sprintf("Multiple occurrence of component for Helidon binding. Invalid Component: [%s]\n", helidonBinding.Name))
		}
	}
	for _, weblogicBinding := range binding.Spec.WeblogicBindings {
		if !componentsInBindingSet[weblogicBinding.Name] {
			componentsInBindingSet[weblogicBinding.Name] = true
		} else {
			errMessages = append(errMessages, fmt.Sprintf("Multiple occurrence of component for Weblogic binding. Invalid Component: [%s]\n", weblogicBinding.Name))
		}
	}

	// Get model referenced in the binding
	modelName := binding.Spec.ModelName
	model, _ := clientsets.V8oClient.VerrazzanoModels(arRequest.Request.Namespace).Get(context.TODO(), modelName, metav1.GetOptions{})

	// Get all components referenced in the model
	componentsInModel := make(map[string]bool)
	for _, coherenceCluster := range model.Spec.CoherenceClusters {
		componentsInModel[coherenceCluster.Name] = true
	}
	for _, helidonApplication := range model.Spec.HelidonApplications {
		componentsInModel[helidonApplication.Name] = true
	}
	for _, weblogicDomain := range model.Spec.WeblogicDomains {
		componentsInModel[weblogicDomain.Name] = true
	}
	for _, genericComponent := range model.Spec.GenericComponents {
		componentsInModel[genericComponent.Name] = true
	}

	// Each componentsInBindingSet component must be present in componentsInModel
	for bindingComponent := range componentsInBindingSet {
		if !componentsInModel[bindingComponent] {
			errMessages = append(errMessages, fmt.Sprintf("Component in bindings does not exist in model definition. Invalid Component: [%s]\n", bindingComponent))
		}
	}

	// Get all components referenced in the placement namespaces
	componentsInPlacementNamespacesSet := make(map[string]bool)

	for _, placement := range binding.Spec.Placement {
		for _, namespace := range placement.Namespaces {
			for _, component := range namespace.Components {
				if !componentsInPlacementNamespacesSet[component.Name] {
					componentsInPlacementNamespacesSet[component.Name] = true
				} else {
					errMessages = append(errMessages, fmt.Sprintf("Multiple occurrence of component across placement namespaces. Invalid Component: [%s]\n", component.Name))
				}
			}
		}
	}
	// Each componentsInPlacementNamespacesSet component must be present in componentsInModel
	for component := range componentsInPlacementNamespacesSet {
		if !componentsInModel[component] {
			errMessages = append(errMessages, fmt.Sprintf("Component in placement namespace does not exist in model definition. Invalid Component: [%s]\n", component))
		}
	}

	if len(errMessages) > 0 {
		glog.Error(s.Join(errMessages, ", "))
	}
	return errMessages
}

// Validate ingressBindings
func validateIngressBinding(ingressBindings []v1beta1v8o.VerrazzanoIngressBinding) []string {
	glog.V(6).Info("In validateIngressBinding code")

	var errMessages []string
	for _, ingressBinding := range ingressBindings {
		// validate ingressBinding > dnsName
		dnsName := s.TrimSpace(ingressBinding.DnsName)
		errFound := false

		// Special case for Verrazzano binding definition where we consider a single * for dnsName as valid.
		if dnsName == "*" {
			continue
		}

		if s.HasPrefix(dnsName, "*.") {
			for _, msg := range k8sValidations.IsWildcardDNS1123Subdomain(dnsName) {
				errMessages = append(errMessages, msg)
				errFound = true
			}
		} else {
			for _, msg := range k8sValidations.IsDNS1123Subdomain(dnsName) {
				errMessages = append(errMessages, msg)
				errFound = true
			}
		}

		if !errFound {
			// Validate labels in the DNS name.
			labels := s.Split(dnsName, ".")
			for i := range labels {
				label := labels[i]
				for _, msg := range k8sValidations.IsDNS1123Label(label) {
					errMessages = append(errMessages, msg)
					errFound = true
				}
			}
		}

		if errFound {
			errMessages = append(errMessages, fmt.Sprintf("Invalid DNS name: [%s]\n", dnsName))
			glog.Error(s.Join(errMessages, ", "))
		}
	}
	return errMessages
}

// Validate that each placement name has a matching VerrazzanoManagedClusters custom resource
func validateClusters(arRequest v1beta1.AdmissionReview, binding v1beta1v8o.VerrazzanoBinding, clientsets *Clientsets) string {
	glog.V(6).Info("In validateClusters code")

	var missingClusters = ""
	for _, placement := range binding.Spec.Placement {
		_, err := clientsets.V8oClient.VerrazzanoManagedClusters(arRequest.Request.Namespace).Get(context.TODO(), placement.Name, metav1.GetOptions{})
		if k8sErrors.IsNotFound(err) {
			if missingClusters != "" {
				missingClusters += ","
			}
			missingClusters += placement.Name
		} else if err != nil {
			message := fmt.Sprintf("failed to get referenced cluster %s in namespace %s: %v", placement.Name, arRequest.Request.Namespace, err)
			glog.Error(message)
			return message
		}
	}

	var message = ""
	if missingClusters != "" {
		message = fmt.Sprintf("binding references cluster(s) \"%s\" that do not exist in namespace %s", missingClusters, arRequest.Request.Namespace)
		glog.Error(message)
	}

	return message
}

// Validate that each secret in the binding has a matching secret in the default namespace
func validateBindingSecrets(binding v1beta1v8o.VerrazzanoBinding, clientsets *Clientsets) string {
	glog.V(6).Info("In validateBindingSecrets code")

	// Check database credentials
	for _, dbBinding := range binding.Spec.DatabaseBindings {
		message := getBindingSecrets(clientsets, dbBinding.Credentials, "databaseBindings.credentials", dbBinding.Name)
		if message != "" {
			return message
		}
	}

	return ""
}

// Get a secret and check for errors
func getBindingSecrets(clientsets *Clientsets, secretName string, secretType string, compName string) string {
	glog.V(6).Info("In getBindingSecrets code")

	_, err := clientsets.K8sClient.CoreV1().Secrets("default").Get(context.TODO(), secretName, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		message := fmt.Sprintf("binding references %s \"%s\" for %s.  This secret must be created in the default namespace before proceeding.", secretType, secretName, compName)
		glog.Error(message)
		return message
	}
	if err != nil {
		message := fmt.Sprintf("failed to get referenced secret %s in namespace default: %v", secretName, err)
		glog.Error(message)
		return message
	}

	return ""
}
