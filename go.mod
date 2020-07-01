module github.com/verrazzano/verrazzano-admission-controllers

go 1.13

require (
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/rs/zerolog v1.19.0
	github.com/verrazzano/verrazzano-crd-generator v0.3.29
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da // indirect
	google.golang.org/protobuf v1.24.0 // indirect
	k8s.io/api v0.18.2
	k8s.io/apiextensions-apiserver v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
)

replace k8s.io/client-go => k8s.io/client-go v0.18.2
