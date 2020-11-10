// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v8oclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/typed/verrazzano/v1beta1"
	"go.uber.org/zap"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ServerHandler listens to admission requests and sends responses
type ServerHandler struct {
	VerrazzanoURI string
}

// Clientsets contains the clients for needed APIs
type Clientsets struct {
	V8oClient v8oclientset.VerrazzanoV1beta1Interface
	K8sClient kubernetes.Interface
}

// Serve function receives validation requests for Verrazzano model and bindings
func (sh *ServerHandler) Serve(w http.ResponseWriter, r *http.Request) {
	zap.S().Infow("Received validation request")

	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		zap.S().Errorw("empty request body received")
		http.Error(w, "empty request body received", http.StatusBadRequest)
		return
	}

	if r.URL.Path != "/validate" {
		zap.S().Errorf("URL prefix %s is not valid", r.URL.Path)
		http.Error(w, fmt.Sprintf("URL prefix %s is not valid", r.URL.Path), http.StatusBadRequest)
		return
	}

	arRequest := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &arRequest); err != nil {
		zap.S().Errorf("error with unmarshal of request body: %v", err)
		http.Error(w, fmt.Sprintf("error with unmarshal of request body: %v", err), http.StatusBadRequest)
		return
	}

	zap.S().Infof("%s operation requested on resource %s", arRequest.Request.Operation, arRequest.Request.Kind.Kind)
	zap.S().Debugf("REQUEST: %+v", arRequest.Request)

	var arResponse = v1beta1.AdmissionReview{}

	clientsets, err := createClientsets()
	if err != nil {
		message := fmt.Sprintf("error getting clientsets: %v", err)
		zap.S().Errorw(message)
		arResponse = v1beta1.AdmissionReview{
			Response: &v1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: message,
				},
			},
		}
	} else {
		switch arRequest.Request.Kind.Kind {
		case "VerrazzanoModel":
			if arRequest.Request.Operation != v1beta1.Delete {
				model := v1beta1v8o.VerrazzanoModel{}
				if err := json.Unmarshal(arRequest.Request.Object.Raw, &model); err != nil {
					zap.S().Errorf("error with unmarshal of VerrazzanoModel: %v", err)
					arResponse = v1beta1.AdmissionReview{
						Response: &v1beta1.AdmissionResponse{
							Allowed: false,
							Result: &metav1.Status{
								Message: fmt.Sprintf("error with unmarshal of VerrazzanoModel: %v", err),
							},
						},
					}
					break
				}
				zap.S().Infof("processing model name: %s:%s", model.Namespace, model.Name)
				arResponse = validateModel(model, clientsets)
			} else {
				zap.S().Infof("processing model name: %s:%s", arRequest.Request.Namespace, arRequest.Request.Name)
				arResponse = deleteModel(arRequest, clientsets)
			}
		case "VerrazzanoBinding":
			binding := v1beta1v8o.VerrazzanoBinding{}
			if err := json.Unmarshal(arRequest.Request.Object.Raw, &binding); err != nil {
				zap.S().Errorf("error with unmarshal of VerrazzanoBinding: %v", err)
				arResponse = v1beta1.AdmissionReview{
					Response: &v1beta1.AdmissionResponse{
						Allowed: false,
						Result: &metav1.Status{
							Message: fmt.Sprintf("error with unmarshal of VerrazzanoBinding: %v", err),
						},
					},
				}
				break
			}
			zap.S().Infof("processing binding name: %s:%s", binding.Namespace, binding.Name)
			arResponse = validateBinding(arRequest, binding, clientsets, sh.VerrazzanoURI)
		default:
			zap.S().Errorf("invalid resource kind %s specified", arRequest.Request.Kind.Kind)
			http.Error(w, fmt.Sprintf("invalid resource kind %s specified", arRequest.Request.Kind.Kind), http.StatusBadRequest)
			return
		}
	}

	// Request was fine so indicate admission request was permitted
	if arResponse.Size() == 0 {
		arResponse = v1beta1.AdmissionReview{
			Response: &v1beta1.AdmissionResponse{
				Allowed: true},
		}
	}

	// Copy the request UID to the response UID
	arResponse.Response.UID = arRequest.Request.UID

	resp, err := json.Marshal(arResponse)
	if err != nil {
		zap.S().Errorf("error with marshal of response: %v", err)
		http.Error(w, fmt.Sprintf("error with marshal of response: %v", err), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		zap.S().Errorf("error with write of response: %v", err)
		http.Error(w, fmt.Sprintf("error with write of response: %v", err), http.StatusInternalServerError)
	}
}

func createClientsets() (*Clientsets, error) {
	zap.S().Debugw("Building kubeconfig")
	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		zap.S().Errorf("Error building kubeconfig: %v", err)
		return nil, err
	}

	zap.S().Debugw("Building Verrazzano clientset")
	v8oclient, err := v8oclientset.NewForConfig(cfg)
	if err != nil {
		zap.S().Errorf("Error building Verrazzano clientset: %v", err)
		return nil, err
	}

	zap.S().Debugw("Building kubernetes clientset")
	k8sclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		zap.S().Errorf("Error building kubernetes clientset: %v", err)
		return nil, err
	}

	return &Clientsets{
		V8oClient: v8oclient,
		K8sClient: k8sclient,
	}, nil
}
