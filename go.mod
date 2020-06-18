module github.com/verrazzano/verrazzano-admission-controllers

go 1.13

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/verrazzano/verrazzano-crd-generator v0.3.27
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da // indirect
	google.golang.org/protobuf v1.24.0 // indirect
	k8s.io/api v0.15.7
	k8s.io/apiextensions-apiserver v0.15.7
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/kind v0.7.0 // indirect
)

// Pinned to kubernetes-0.15.7
replace (
	k8s.io/api => k8s.io/api v0.15.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.15.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.15.7
	k8s.io/apiserver => k8s.io/apiserver v0.15.7
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.15.7
	k8s.io/client-go => k8s.io/client-go v0.15.7
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.15.7
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.15.7
	k8s.io/code-generator => k8s.io/code-generator v0.15.7
	k8s.io/component-base => k8s.io/component-base v0.15.7
	k8s.io/cri-api => k8s.io/cri-api v0.15.7
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.15.7
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.15.7
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.15.7
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.15.7
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.15.7
	k8s.io/kubelet => k8s.io/kubelet v0.15.7
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.15.7
	k8s.io/metrics => k8s.io/metrics v0.15.7
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.15.7
)
