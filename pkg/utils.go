// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sValidations "k8s.io/apimachinery/pkg/util/validation"
)

const invalidNameFormat = "\n* %s: Invalid value: \"%s\": %s"

// Add invalidNameFormat message to list of error messages.
func addInvalidNameFormatMessage(name string, field string, errMessages []string) []string {

	for _, msg := range k8sValidations.IsDNS1123Subdomain(name) {
		msgOut := fmt.Sprintf(invalidNameFormat, field, name, msg)
		glog.Error(msgOut)
		errMessages = append(errMessages, msgOut)
	}

	return errMessages
}

// Create an error response
func errorAdmissionReview(errMessage string) v1beta1.AdmissionReview {
	return v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: errMessage,
			},
		},
	}
}
