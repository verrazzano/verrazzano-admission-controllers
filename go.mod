module github.com/verrazzano/verrazzano-admission-controllers

go 1.13

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/spf13/pflag v1.0.5
	github.com/verrazzano/verrazzano-crd-generator v0.3.29
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da // indirect
	google.golang.org/protobuf v1.24.0 // indirect
	k8s.io/api v0.18.2
	k8s.io/apiextensions-apiserver v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/kind v0.7.0 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.18.2
