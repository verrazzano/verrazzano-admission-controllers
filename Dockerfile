# Copyright (C) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM container-registry.oracle.com/os/oraclelinux:7-slim@sha256:fcc6f54bb01fc83319990bf5fa1b79f1dec93cbb87db3c5a8884a5a44148e7bb AS build_base

RUN yum update -y \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum install -y oracle-golang-release-el7 \
    && yum-config-manager --add-repo http://yum.oracle.com/repo/OracleLinux/OL7/developer/golang113/x86_64 \
    && yum install -y git gcc make golang-1.13.3-1.el7 \
    && yum clean all \
    && go version

# Compile to /usr/bin
ENV GOBIN=/usr/bin

# Set go path
ENV GOPATH=/go

ARG BUILDVERSION
ARG BUILDDATE

WORKDIR /go/src/github.com/verrazzano/verrazzano-admission-controllers
COPY . .

ENV CGO_ENABLED 0
RUN go version
RUN go env

RUN GO111MODULE=on go build \
    -mod=vendor \
    -ldflags '-extldflags "-static"' \
    -ldflags "-X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}" \
    -o /usr/bin/verrazzano-admission-controller ./cmd/...

FROM container-registry.oracle.com/os/oraclelinux:7-slim@sha256:fcc6f54bb01fc83319990bf5fa1b79f1dec93cbb87db3c5a8884a5a44148e7bb

RUN yum update -y \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum install -y ca-certificates curl openssl \
    && yum clean all \
    && rm -rf /var/cache/yum

COPY --from=build_base /usr/bin/verrazzano-admission-controller /usr/local/bin/verrazzano-admission-controller

RUN groupadd -r verrazzano-admission-controller && useradd --no-log-init -r -g verrazzano-admission-controller -u 1000 verrazzano-admission-controller
RUN chown 1000:verrazzano-admission-controller /usr/local/bin/verrazzano-admission-controller && chmod 500 /usr/local/bin/verrazzano-admission-controller
USER 1000

ENTRYPOINT ["/usr/local/bin/verrazzano-admission-controller"]
