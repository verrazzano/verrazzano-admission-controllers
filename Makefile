# Copyright (C) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

NAME:=verrazzano-admission-controller

DOCKER_IMAGE_NAME ?= ${NAME}-dev
TAG=$(shell git rev-parse HEAD)
DOCKER_IMAGE_TAG = ${TAG}
SHORT_COMMIT_HASH=$(shell git rev-parse --short HEAD)
PUBLISH_IMAGE_TAG = ${TAG_NAME}-${SHORT_COMMIT_HASH}-${BUILD_NUMBER}

CREATE_LATEST_TAG=0

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),push push-tag))
	ifndef DOCKER_REPO
		$(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
	endif
	ifndef DOCKER_NAMESPACE
		$(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
	endif
	DOCKER_IMAGE_FULLNAME = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${DOCKER_IMAGE_NAME}
endif

GO ?= GO111MODULE=on GOPRIVATE=github.com/oracle,github.com/verrazzano go

CLUSTER_NAME = admission-controller
GITHUB_PATH = github.com/verrazzano
CRDGEN_PATH = ${GITHUB_PATH}/verrazzano-crd-generator
CRD_PATH = deploy/crds
CERTS = build/admission-controller-cert
VERRAZZANO_NS = verrazzano-system
DEPLOY = build/deploy

.PHONY: all
all: build

#
# Go build related tasks
#
.PHONY: go-install
go-install: go-mod
	$(GO) install ./cmd/...

.PHONY: go-fmt
go-fmt:
	gofmt -s -e -d $(shell find . -name "*.go" | grep -v /vendor/)

.PHONY: go-mod
go-mod:
	$(GO) mod vendor

	# Obtain verrazzano-crd-generator version
	mkdir -p vendor/${CRDGEN_PATH}/${CRD_PATH}
	cp `go list -f '{{.Dir}}' -m github.com/verrazzano/verrazzano-crd-generator`/${CRD_PATH}/*.yaml vendor/${CRDGEN_PATH}/${CRD_PATH}

	# List copied CRD YAMLs
	ls vendor/${CRDGEN_PATH}/${CRD_PATH}

.PHONY: build
build: go-mod
	docker build \
		-t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: push
push: build
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	docker push ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}

	if [ "${CREATE_LATEST_TAG}" == "1" ]; then \
		docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:latest; \
		docker push ${DOCKER_IMAGE_FULLNAME}:latest; \
	fi


.PHONY: push-tag
push-tag:
	docker pull ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	docker tag ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:${PUBLISH_IMAGE_TAG}
	docker push ${DOCKER_IMAGE_FULLNAME}:${PUBLISH_IMAGE_TAG}

#
# Tests-related tasks
#
.PHONY: unit-test
unit-test: go-install
	go test -v ./pkg/apis/... ./pkg/controler/... ./cmd/...

.PHONY: coverage
coverage:
	./build/scripts/coverage.sh html

#
# Tests-related tasks
#
.PHONY: integ-test
integ-test: build create-cluster
	echo 'Load docker image for the admission-controller...'
	kind load docker-image --name ${CLUSTER_NAME} ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

	echo 'Create resources needed by the admission-controller...'
	kubectl create -f vendor/${CRDGEN_PATH}/${CRD_PATH}/verrazzano.io_verrazzanomanagedclusters_crd.yaml
	kubectl create -f vendor/${CRDGEN_PATH}/${CRD_PATH}/verrazzano.io_verrazzanomodels_crd.yaml
	kubectl create -f vendor/${CRDGEN_PATH}/${CRD_PATH}/verrazzano.io_verrazzanobindings_crd.yaml

	echo 'Deploy admission controller...'
	kubectl create namespace ${VERRAZZANO_NS}
	./test/certs/create-cert.sh
	kubectl create secret generic verrazzano-validation -n ${VERRAZZANO_NS} \
			--from-file=cert.pem=${CERTS}/verrazzano-crt.pem \
			--from-file=key.pem=${CERTS}/verrazzano-key.pem
	./test/create-deployment.sh ${DOCKER_IMAGE_NAME} ${DOCKER_IMAGE_TAG}
	kubectl apply -f ${DEPLOY}/deployment.yaml

	echo 'Run tests...'
	ginkgo -v --keepGoing -cover test/integ/... || IGNORE=FAILURE

.PHONY: create-cluster
create-cluster:
ifdef JENKINS_URL
	./build/scripts/cleanup.sh ${CLUSTER_NAME}
endif
	echo 'Create cluster...'
	HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster \
		--name ${CLUSTER_NAME} \
		--wait 5m \
		--config=test/kind-config.yaml
	kubectl config set-context kind-${CLUSTER_NAME}
ifdef JENKINS_URL
	# disabled this - not needed since we are not running from inside docker any more
	# cat ${HOME}/.kube/config | grep server
	# this ugly looking line of code will get the ip address of the container running the kube apiserver
	# and update the kubeconfig file to point to that address, instead of localhost
	sed -i -e "s|127.0.0.1.*|`docker inspect ${CLUSTER_NAME}-control-plane | jq '.[].NetworkSettings.IPAddress' | sed 's/"//g'`:6443|g" ${HOME}/.kube/config
	cat ${HOME}/.kube/config | grep server
endif

.PHONY: delete-cluster
delete-cluster:
	kind delete cluster --name ${CLUSTER_NAME}
