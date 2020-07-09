// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
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

func validateModel(model v1beta1v8o.VerrazzanoModel, clientsets *Clientsets) v1beta1.AdmissionReview {
	glog.V(6).Info("In validateModel code")

	// All secrets in the model must be defined in the default namespace.
	response := validateModelSecrets(model, clientsets)
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

	glog.Info("validation of model successful")
	return v1beta1.AdmissionReview{}
}

func deleteModel(arRequest v1beta1.AdmissionReview, clientsets *Clientsets) v1beta1.AdmissionReview {
	glog.V(6).Info("In deleteModel code")

	// Delete is being called for namespaces (for some unknown reason) when there is single cluster.  In this case,
	// there is no resource name so just return and don't generate an error.
	if len(arRequest.Request.Name) == 0 {
		_, err := clientsets.K8sClientset.CoreV1().Namespaces().Get(context.TODO(), arRequest.Request.Namespace, metav1.GetOptions{})
		if err == nil {
			glog.Info("delete of namespace was requested, no model to delete")
			return v1beta1.AdmissionReview{}
		}
	}

	// Get the model we want to delete
	model, err := clientsets.V8oClientset.VerrazzanoV1beta1().VerrazzanoModels(arRequest.Request.Namespace).Get(context.TODO(), arRequest.Request.Name, metav1.GetOptions{})

	// Delete is called for resources that don't exist. If that is the case, then just return
	if k8sErrors.IsNotFound(err) {
		glog.Info("model does not exist, nothing to delete")
		return v1beta1.AdmissionReview{}
	}

	// Don't allow delete if we had an error getting the model
	if err != nil {
		message := fmt.Sprintf("error getting model for namespace %s: %v", arRequest.Request.Namespace, err)
		glog.Error(message)
		return errorAdmissionReview(message)
	}

	// Don't allow delete if a deployed binding references this model
	if model != nil {
		bindingList, err := clientsets.V8oClientset.VerrazzanoV1beta1().VerrazzanoBindings(arRequest.Request.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err == nil && bindingList != nil {
			for _, binding := range bindingList.Items {
				if binding.Spec.ModelName == model.Name {
					message := fmt.Sprintf("model cannot be deleted before binding %s is deleted in namespace %s", binding.Name, arRequest.Request.Namespace)
					glog.Error(message)
					return errorAdmissionReview(message)
				}
			}
		}
	}

	glog.Info("validation of model successful")
	return v1beta1.AdmissionReview{}
}

// Validate that each secret in the model has a matching secret in the default namespace
func validateModelSecrets(model v1beta1v8o.VerrazzanoModel, clientsets *Clientsets) string {
	glog.V(6).Info("In validateModelSecrets code")

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

	// Check image pull secrets for Weblogic domains
	for _, domain := range model.Spec.WeblogicDomains {
		for _, secret := range domain.DomainCRValues.ImagePullSecrets {
			message := getSecret(clientsets, secret.Name, "weblogicDomains.domainCRValues.imagePullSecret", domain.Name)
			if message != "" {
				return message
			}
		}
	}

	// Check Weblogic domain credential secrets
	for _, cred := range model.Spec.WeblogicDomains {
		secret := cred.DomainCRValues.WebLogicCredentialsSecret
		message := getSecret(clientsets, secret.Name, "weblogicDomains.domainCRValues.webLogicCredentialsSecret", cred.Name)
		if message != "" {
			return message
		}
	}

	// Check Weblogic domain config override secrets
	for _, configOverride := range model.Spec.WeblogicDomains {
		for _, secret := range configOverride.DomainCRValues.ConfigOverrideSecrets {
			message := getSecret(clientsets, secret, "weblogicDomains.domainCRValues.configOverrideSecrets", configOverride.Name)
			if message != "" {
				return message
			}
		}
	}

	return ""
}

// Get a secret and check for errors
func getSecret(clientsets *Clientsets, secretName string, secretType string, compName string) string {
	glog.V(6).Info("In getSecret code")

	_, err := clientsets.K8sClientset.CoreV1().Secrets("default").Get(context.TODO(), secretName, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		message := fmt.Sprintf("model references %s \"%s\" for component %s.  This secret must be created in the default namespace before proceeding.", secretType, secretName, compName)
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

func validateCoherenceClusters(model v1beta1v8o.VerrazzanoModel) string {
	glog.V(6).Info("In validateCoherenceClusters code")

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

func validateWebLogicDomains(model v1beta1v8o.VerrazzanoModel) string {
	glog.V(6).Info("In validateWebLogicDomains code")

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
			message := fmt.Sprintf("AdminPort and T3Port in Weblogic domain %s have the same value: %v", wd.Name, wd.AdminPort)
			glog.Error(message)
			portMessages = append(portMessages, message)
		}

		if len(portMessages) > 0 {
			return s.Join(portMessages, "; ")
		}
	}
	return ""
}

func validateHelidonApplications(model v1beta1v8o.VerrazzanoModel) string {
	glog.V(6).Info("In validateHelidonApplications code")

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
			glog.Error(errors)
			return errors
		}
		errMessages = k8sValidations.IsEnvVarName(rc.EnvironmentVariableForPort)
		if len(errMessages) > 0 {
			errMessages = append(errMessages, fmt.Sprintf("Invalid variable name: %s", rc.EnvironmentVariableForPort))
			errors := s.Join(errMessages, ", ")
			glog.Error(errors)
			return errors
		}
		if rc.EnvironmentVariableForPort == rc.EnvironmentVariableForHost {
			message := fmt.Sprintf("REST connection for target %s uses the same environment variable for host and port: %s", rc.Target, rc.EnvironmentVariableForHost)
			glog.Error(message)
			return message
		}
	}
	return ""
}

func validatePort(port int) string {
	glog.V(6).Info("Received this port: ", port)
	errMessages := k8sValidations.IsValidPortNum(port)
	if len(errMessages) > 0 {
		invalidPortMsg := fmt.Sprintf("Port %v is not valid. ", port)
		errors := invalidPortMsg + s.Join(errMessages, ", ")
		glog.Error(errors)
		return errors
	}
	return ""
}
