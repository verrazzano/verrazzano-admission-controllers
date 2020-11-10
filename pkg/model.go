// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	s "strings"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"go.uber.org/zap"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sValidations "k8s.io/apimachinery/pkg/util/validation"
)

func validateModel(model v1beta1v8o.VerrazzanoModel, clientsets *Clientsets) v1beta1.AdmissionReview {
	zap.S().Debugw("In validateModel code")

	response := validateModelResourceNames(model)
	if response != "" {
		return errorAdmissionReview(response)
	}

	response = validateSingleWebLogicCluster(model)
	if response != "" {
		return errorAdmissionReview(response)
	}

	// All secrets in the model must be defined in the default namespace.
	response = validateModelSecrets(model, clientsets)
	if response != "" {
		return errorAdmissionReview(response)
	}

	response = validateWebLogicDomains(model)
	if response != "" {
		return errorAdmissionReview(response)
	}

	response = validateCoherenceClusters(model)
	if response != "" {
		return errorAdmissionReview(response)
	}

	response = validateHelidonApplications(model)
	if response != "" {
		return errorAdmissionReview(response)
	}

	response = validateGenericComponents(model)
	if response != "" {
		return errorAdmissionReview(response)
	}

	zap.S().Infow("validation of model successful")
	return v1beta1.AdmissionReview{}
}

func deleteModel(arRequest v1beta1.AdmissionReview, clientsets *Clientsets) v1beta1.AdmissionReview {
	zap.S().Debugw("In deleteModel code")

	// Delete is being called for namespaces (for some unknown reason) when there is single cluster.  In this case,
	// there is no resource name so just return and don't generate an error.
	if len(arRequest.Request.Name) == 0 {
		_, err := clientsets.K8sClient.CoreV1().Namespaces().Get(context.TODO(), arRequest.Request.Namespace, metav1.GetOptions{})
		if err == nil {
			zap.S().Infow("delete of namespace was requested, no model to delete")
			return v1beta1.AdmissionReview{}
		}
	}

	// Get the model we want to delete
	model, err := clientsets.V8oClient.VerrazzanoModels(arRequest.Request.Namespace).Get(context.TODO(), arRequest.Request.Name, metav1.GetOptions{})

	// Delete is called for resources that don't exist. If that is the case, then just return
	if k8sErrors.IsNotFound(err) {
		zap.S().Infow("model does not exist, nothing to delete")
		return v1beta1.AdmissionReview{}
	}

	// Don't allow delete if we had an error getting the model
	if err != nil {
		message := fmt.Sprintf("error getting model for namespace %s: %v", arRequest.Request.Namespace, err)
		zap.S().Errorw(message)
		return errorAdmissionReview(message)
	}

	// Don't allow delete if a deployed binding references this model
	if model != nil {
		bindingList, err := clientsets.V8oClient.VerrazzanoBindings(arRequest.Request.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err == nil && bindingList != nil {
			for _, binding := range bindingList.Items {
				if binding.Spec.ModelName == model.Name {
					message := fmt.Sprintf("model cannot be deleted before binding %s is deleted in namespace %s", binding.Name, arRequest.Request.Namespace)
					zap.S().Errorw(message)
					return errorAdmissionReview(message)
				}
			}
		}
	}

	zap.S().Infow("validation of model successful")
	return v1beta1.AdmissionReview{}
}

// Validate names that will be used as Kubernetes resource names.
// A validate k8s resource name must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an
// alphanumeric character.  We use k8s validation functions to check the validity of names.
func validateModelResourceNames(model v1beta1v8o.VerrazzanoModel) string {
	zap.S().Debugw("In validateModelResourceNames code")

	var errMessages []string

	for _, msg := range validateModelHelidonNames(model) {
		errMessages = append(errMessages, msg)
	}

	for _, msg := range validateModelCoherenceNames(model) {
		errMessages = append(errMessages, msg)
	}

	for _, msg := range validateModelWeblogicNames(model) {
		errMessages = append(errMessages, msg)
	}

	for _, msg := range validateModelGenericComponentNames(model) {
		errMessages = append(errMessages, msg)
	}

	for _, msg := range validateModelAllIngressNames(model) {
		errMessages = append(errMessages, msg)
	}

	if len(errMessages) > 0 {
		return s.Join(errMessages, "")
	}

	return ""
}

// Validate names for Helidon applications
func validateModelHelidonNames(model v1beta1v8o.VerrazzanoModel) []string {
	zap.S().Debugw("In validateModelHelidonNames code")

	var errMessages []string

	for i, ha := range model.Spec.HelidonApplications {
		// Check the Helidon component name
		field := fmt.Sprintf("spec.helidonApplications[%d].name", i)
		errMessages = addInvalidNameFormatMessage(ha.Name, field, errMessages)

		// Check the Helidon imagePullSecrets name
		for k, secret := range ha.ImagePullSecrets {
			field := fmt.Sprintf("spec.helidonApplications[%d].imagePullSecrets[%d].name", i, k)
			errMessages = addInvalidNameFormatMessage(secret.Name, field, errMessages)
		}
	}

	return errMessages
}

// Validate names for Coherence clusters
func validateModelCoherenceNames(model v1beta1v8o.VerrazzanoModel) []string {
	zap.S().Debugw("In validateModelCoherenceNames code")

	var errMessages []string

	for i, cc := range model.Spec.CoherenceClusters {
		// Check the Coherence component name
		field := fmt.Sprintf("spec.coherenceClusters[%d].name", i)
		errMessages = addInvalidNameFormatMessage(cc.Name, field, errMessages)

		// Check the Coherence imagePullSecrets name
		for k, secret := range cc.ImagePullSecrets {
			field := fmt.Sprintf("spec.coherenceClusters[%d].imagePullSecrets[%d].name", i, k)
			errMessages = addInvalidNameFormatMessage(secret.Name, field, errMessages)
		}
	}

	return errMessages
}

// Validate names for WebLogic domains
func validateModelWeblogicNames(model v1beta1v8o.VerrazzanoModel) []string {
	zap.S().Debugw("In validateModelWeblogicNames code")

	var errMessages []string

	for i, domain := range model.Spec.WeblogicDomains {
		// Check the WebLogic component name
		field := fmt.Sprintf("spec.weblogicDomains[%d].name", i)
		errMessages = addInvalidNameFormatMessage(domain.Name, field, errMessages)

		// Check the WebLogic domain UID name
		if len(domain.DomainCRValues.DomainUID) > 0 {
			field := fmt.Sprintf("spec.weblogicDomains[%d].domainCRValues.domainUID", i)
			errMessages = addInvalidNameFormatMessage(domain.DomainCRValues.DomainUID, field, errMessages)
		}

		// Check the WebLogic imagePullSecrets name
		for j, secret := range domain.DomainCRValues.ImagePullSecrets {
			field := fmt.Sprintf("spec.weblogicDomains[%d].domainCRValues.imagePullSecrets[%d].name", i, j)
			errMessages = addInvalidNameFormatMessage(secret.Name, field, errMessages)
		}

		// Check the webLogicCredentialsSecret name
		secret := domain.DomainCRValues.WebLogicCredentialsSecret
		field = fmt.Sprintf("spec.weblogicDomains[%d].domainCRValues.webLogicCredentialsSecret.name", i)
		errMessages = addInvalidNameFormatMessage(secret.Name, field, errMessages)

		// Check the WebLogic configOverrideSecrets name
		for j, secret := range domain.DomainCRValues.ConfigOverrideSecrets {
			field := fmt.Sprintf("spec.weblogicDomains[%d].domainCRValues.configOverrideSecrets[%d]", i, j)
			errMessages = addInvalidNameFormatMessage(secret, field, errMessages)
		}

		// Check the WebLogic configuration secrets name
		for j, secret := range domain.DomainCRValues.Configuration.Secrets {
			field := fmt.Sprintf("spec.weblogicDomains[%d].domainCRValues.configuration.secrets[%d]", i, j)
			errMessages = addInvalidNameFormatMessage(secret, field, errMessages)
		}
	}

	return errMessages
}

// Validate names for generic components
func validateModelGenericComponentNames(model v1beta1v8o.VerrazzanoModel) []string {
	zap.S().Debugw("In validateModelGenericComponentNames code")

	var errMessages []string

	for i, generic := range model.Spec.GenericComponents {
		// Check the generic component name
		field := fmt.Sprintf("spec.genericComponents[%d].name", i)
		errMessages = addInvalidNameFormatMessage(generic.Name, field, errMessages)

		// Check the generic component imagePullSecrets name
		for j, secret := range generic.Deployment.ImagePullSecrets {
			field := fmt.Sprintf("spec.genericComponents[%d].deployment.imagePullSecrets[%d].name", i, j)
			errMessages = addInvalidNameFormatMessage(secret.Name, field, errMessages)
		}

		// Check the generic component deployment containers for secret name references
		for j, container := range generic.Deployment.Containers {
			for k, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					field := fmt.Sprintf("spec.genericComponents[%d].deployment.containers[%d].env[%d].valueFrom.secretKeyRef.name", i, j, k)
					errMessages = addInvalidNameFormatMessage(env.ValueFrom.SecretKeyRef.Name, field, errMessages)
				}
			}
		}

		// Check the generic component deployment init containers for secret name references
		for j, container := range generic.Deployment.InitContainers {
			for k, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					field := fmt.Sprintf("spec.genericComponents[%d].deployment.initContainers[%d].env[%d].valueFrom.secretKeyRef.name", i, j, k)
					errMessages = addInvalidNameFormatMessage(env.ValueFrom.SecretKeyRef.Name, field, errMessages)
				}
			}
		}

	}

	return errMessages
}

// Validate ingress connection names for all components
func validateModelAllIngressNames(model v1beta1v8o.VerrazzanoModel) []string {
	zap.S().Debugw("In validateModelAllIngressNames code")

	var errMessages []string

	// Check the Helidon applications ingress names
	for i, ha := range model.Spec.HelidonApplications {
		for j, connection := range ha.Connections {
			prefix := fmt.Sprintf("spec.helidonApplications[%d].connections[%d]", i, j)
			for _, msg := range validateModelIngressNames(connection.Ingress, prefix) {
				errMessages = append(errMessages, msg)
			}
		}
	}

	// Check the Coherence clusters ingress names
	for i, cc := range model.Spec.CoherenceClusters {
		for j, connection := range cc.Connections {
			prefix := fmt.Sprintf("spec.coherenceClusters[%d].connections[%d]", i, j)
			for _, msg := range validateModelIngressNames(connection.Ingress, prefix) {
				errMessages = append(errMessages, msg)
			}
		}
	}

	// Check the WebLogic domains ingress names
	for i, domain := range model.Spec.WeblogicDomains {
		for j, connection := range domain.Connections {
			prefix := fmt.Sprintf("spec.weblogicDomains[%d].connections[%d]", i, j)
			for _, msg := range validateModelIngressNames(connection.Ingress, prefix) {
				errMessages = append(errMessages, msg)
			}
		}
	}

	// Check the generic components ingress names
	for i, generic := range model.Spec.GenericComponents {
		for j, connection := range generic.Connections {
			prefix := fmt.Sprintf("spec.genericComponents[%d].connections[%d]", i, j)
			for _, msg := range validateModelIngressNames(connection.Ingress, prefix) {
				errMessages = append(errMessages, msg)
			}
		}
	}

	return errMessages
}

// Validate ingress connections names
func validateModelIngressNames(connections []v1beta1v8o.VerrazzanoIngressConnection, prefix string) []string {
	zap.S().Debugw("In validateModelIngressNames code")

	var errMessages []string

	for i, ingress := range connections {
		field := fmt.Sprintf("%s.ingress[%d].name", prefix, i)
		errMessages = addInvalidNameFormatMessage(ingress.Name, field, errMessages)
	}

	return errMessages
}

// Validate that each secret in the model has a matching secret in the default namespace
func validateModelSecrets(model v1beta1v8o.VerrazzanoModel, clientsets *Clientsets) string {
	zap.S().Debugw("In validateModelSecrets code")

	// Check image pull secrets for Helidon applications
	for _, ha := range model.Spec.HelidonApplications {
		for _, secret := range ha.ImagePullSecrets {
			message := getSecret(clientsets, secret.Name, "helidonApplications.imagePullSecret", ha.Name)
			if message != "" {
				return message
			}
		}
	}

	// Check image pull secrets for Coherence clusters
	for _, cc := range model.Spec.CoherenceClusters {
		for _, secret := range cc.ImagePullSecrets {
			message := getSecret(clientsets, secret.Name, "coherenceClusters.imagePullSecret", cc.Name)
			if message != "" {
				return message
			}
		}
	}

	// Check image pull secrets for WebLogic domains
	for _, domain := range model.Spec.WeblogicDomains {
		for _, secret := range domain.DomainCRValues.ImagePullSecrets {
			message := getSecret(clientsets, secret.Name, "weblogicDomains.domainCRValues.imagePullSecret", domain.Name)
			if message != "" {
				return message
			}
		}
	}

	// Check WebLogic domain credential secrets
	for _, cred := range model.Spec.WeblogicDomains {
		secret := cred.DomainCRValues.WebLogicCredentialsSecret
		message := getSecret(clientsets, secret.Name, "weblogicDomains.domainCRValues.webLogicCredentialsSecret", cred.Name)
		if message != "" {
			return message
		}
	}

	// Check WebLogic domain config override secrets
	for _, configOverride := range model.Spec.WeblogicDomains {
		for _, secret := range configOverride.DomainCRValues.ConfigOverrideSecrets {
			message := getSecret(clientsets, secret, "weblogicDomains.domainCRValues.configOverrideSecrets", configOverride.Name)
			if message != "" {
				return message
			}
		}
	}

	// Check WebLogic domain configuration secrets
	for _, configurationSecrets := range model.Spec.WeblogicDomains {
		for _, secret := range configurationSecrets.DomainCRValues.Configuration.Secrets {
			message := getSecret(clientsets, secret, "weblogicDomains.domainCRValues.configuration.secrets", configurationSecrets.Name)
			if message != "" {
				return message
			}
		}
	}

	// Check GenericComponents' secrets
	for _, gc := range model.Spec.GenericComponents {
		for _, sec := range gc.Deployment.ImagePullSecrets {
			message := getSecret(clientsets, sec.Name, "genericComponents.Deployment.Template.Spec.ImagePullSecrets", gc.Name)
			if message != "" {
				return message
			}
		}
		for _, container := range gc.Deployment.InitContainers {
			message := validateContainerEnv(container, "genericComponents.Deployment.InitContainers.Env", gc.Name, clientsets)
			if message != "" {
				return message
			}
		}
		for _, container := range gc.Deployment.Containers {
			message := validateContainerEnv(container, "genericComponents.Deployment.Containers.Env", gc.Name, clientsets)
			if message != "" {
				return message
			}
		}
	}

	return ""
}

// Get a secret and check for errors
func getSecret(clientsets *Clientsets, secretName string, secretType string, compName string) string {
	zap.S().Debugw("In getSecret code")

	_, err := clientsets.K8sClient.CoreV1().Secrets("default").Get(context.TODO(), secretName, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		message := fmt.Sprintf("model references %s \"%s\" for component %s.  This secret must be created in the default namespace before proceeding.", secretType, secretName, compName)
		zap.S().Errorw(message)
		return message
	}
	if err != nil {
		message := fmt.Sprintf("failed to get referenced secret %s in namespace default: %v", secretName, err)
		zap.S().Errorw(message)
		return message
	}

	return ""
}

func validateCoherenceClusters(model v1beta1v8o.VerrazzanoModel) string {
	zap.S().Debugw("In validateCoherenceClusters code")

	for _, cc := range model.Spec.CoherenceClusters {
		for _, connection := range cc.Connections {
			message := validateRestConnections(connection.Rest)
			if message != "" {
				return message
			}
		}
	}
	return ""
}

// Validate that there is only one WebLogic cluster per domain
func validateSingleWebLogicCluster(model v1beta1v8o.VerrazzanoModel) string {
	zap.S().Debugw("In validateSingleWebLogicCluster code")

	var messages []string
	for _, wd := range model.Spec.WeblogicDomains {
		if len(wd.DomainCRValues.Clusters) > 1 {
			message := fmt.Sprintf("More than one WebLogic cluster is not allowed for WebLogic domain %s", wd.Name)
			zap.S().Errorw(message)
			messages = append(messages, message)
		}
	}

	if len(messages) > 0 {
		return s.Join(messages, "; ")
	}

	return ""
}

func validateWebLogicDomains(model v1beta1v8o.VerrazzanoModel) string {
	zap.S().Debugw("In validateWebLogicDomains code")

	for _, wd := range model.Spec.WeblogicDomains {
		for _, connection := range wd.Connections {
			message := validateRestConnections(connection.Rest)
			if message != "" {
				return message
			}
		}
		var portMessages []string
		if wd.AdminPort != 0 {
			message := validatePort(wd.AdminPort)
			if message != "" {
				portMessages = append(portMessages, message)
			}
		}
		if wd.AdminPort != 0 {
			message := validatePort(wd.T3Port)
			if message != "" {
				portMessages = append(portMessages, message)
			}
		}

		if wd.AdminPort != 0 && wd.T3Port != 0 && wd.AdminPort == wd.T3Port {
			message := fmt.Sprintf("AdminPort and T3Port in WebLogic domain %s have the same value: %v", wd.Name, wd.AdminPort)
			zap.S().Errorw(message)
			portMessages = append(portMessages, message)
		}

		if len(portMessages) > 0 {
			return s.Join(portMessages, "; ")
		}
	}
	return ""
}

func validateHelidonApplications(model v1beta1v8o.VerrazzanoModel) string {
	zap.S().Debugw("In validateHelidonApplications code")

	for _, ha := range model.Spec.HelidonApplications {
		for _, connection := range ha.Connections {
			message := validateRestConnections(connection.Rest)
			if message != "" {
				return message
			}
		}
		var portMessages []string
		if ha.Port != 0 {
			message := validatePort(int(ha.Port))
			if message != "" {
				portMessages = append(portMessages, message)
			}
		}
		if ha.TargetPort != 0 {
			message := validatePort(int(ha.TargetPort))
			if message != "" {
				portMessages = append(portMessages, message)
			}
		}
		if len(portMessages) > 0 {
			return s.Join(portMessages, "; ")
		}
	}
	return ""
}

func validateRestConnections(restConnections []v1beta1v8o.VerrazzanoRestConnection) string {
	for _, rc := range restConnections {
		errMessages := k8sValidations.IsEnvVarName(rc.EnvironmentVariableForHost)
		if len(errMessages) > 0 {
			errMessages = append(errMessages, fmt.Sprintf("Invalid variable name: %s", rc.EnvironmentVariableForHost))
			errors := s.Join(errMessages, ", ")
			zap.S().Errorw(errors)
			return errors
		}
		errMessages = k8sValidations.IsEnvVarName(rc.EnvironmentVariableForPort)
		if len(errMessages) > 0 {
			errMessages = append(errMessages, fmt.Sprintf("Invalid variable name: %s", rc.EnvironmentVariableForPort))
			errors := s.Join(errMessages, ", ")
			zap.S().Errorw(errors)
			return errors
		}
		if rc.EnvironmentVariableForPort == rc.EnvironmentVariableForHost {
			message := fmt.Sprintf("REST connection for target %s uses the same environment variable for host and port: %s", rc.Target, rc.EnvironmentVariableForHost)
			zap.S().Errorw(message)
			return message
		}
	}
	return ""
}

func validatePort(port int) string {
	zap.S().Debugw("Received this port: ", port)
	errMessages := k8sValidations.IsValidPortNum(port)
	if len(errMessages) > 0 {
		invalidPortMsg := fmt.Sprintf("Port %v is not valid. ", port)
		errors := invalidPortMsg + s.Join(errMessages, ", ")
		zap.S().Errorw(errors)
		return errors
	}
	return ""
}

func validateGenericComponents(model v1beta1v8o.VerrazzanoModel) string {
	// Check GenericComponents' secrets
	var errorMessages []string
	for _, gc := range model.Spec.GenericComponents {
		for _, container := range gc.Deployment.InitContainers {
			errorMessages = validateContainerPort(container, errorMessages)
		}
		for _, container := range gc.Deployment.Containers {
			errorMessages = validateContainerPort(container, errorMessages)
		}
		for _, connection := range gc.Connections {
			message := validateRestConnections(connection.Rest)
			if message != "" {
				errorMessages = append(errorMessages, message)
			}
		}
	}
	if len(errorMessages) > 0 {
		return s.Join(errorMessages, "; ")
	}
	return ""
}

func validateContainerEnv(container corev1.Container, secretType, compName string, clientsets *Clientsets) string {
	for _, ev := range container.Env {
		if ev.ValueFrom != nil && ev.ValueFrom.SecretKeyRef != nil {
			secName := ev.ValueFrom.SecretKeyRef.Name
			message := getSecret(clientsets, secName, secretType, compName)
			if message != "" {
				return message
			}
		}
	}
	return ""
}

func validateContainerPort(container corev1.Container, errorMessages []string) []string {
	for _, port := range container.Ports {
		if port.ContainerPort != 0 {
			message := validatePort(int(port.ContainerPort))
			if message != "" {
				errorMessages = append(errorMessages, message)
			}
		}
	}
	return errorMessages
}
