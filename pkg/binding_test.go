// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"testing"
)

// testing validateIngressBinding
func TestValidateIngressBinding(t *testing.T) {
	// testing valid binding instances
	goodBindings := []v1beta1v8o.VerrazzanoIngressBinding{{Name: "binding1", DnsName: "*"}, {Name: "binding2", DnsName: "oracle.com"}}
	if errs := validateIngressBinding(goodBindings); len(errs) > 0 {
		t.Errorf("valid ingress bindings were not validated correctly: %s and %s", goodBindings[0].DnsName, goodBindings[1].DnsName)
	}

	// testing invalid binding instances
	badBindings := []v1beta1v8o.VerrazzanoIngressBinding{{Name: "binding1", DnsName: "*oracle.com"}, {Name: "binding2", DnsName: "oracle@com"}}
	if errs := validateIngressBinding(badBindings); len(errs) == 0 {
		t.Errorf("invalid ingress bindings were falsely validated: %s and %s", badBindings[0].DnsName, badBindings[1].DnsName)
	}

}