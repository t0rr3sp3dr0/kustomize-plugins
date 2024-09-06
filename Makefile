.DEFAULT_GOAL = build
SHELL = /bin/bash

RESET := $(shell tput sgr0)
BOLD := $(shell tput bold)
RED := $(shell tput setaf 1)
EOL := \n

API_GROUP ?= incognia.com
API_VERSION ?= v1alpha1
PLACEMENT ?= $(shell echo $${XDG_CONFIG_HOME:-$$HOME/.config}/kustomize/plugin/${API_GROUP}/${API_VERSION})

setup-environment:
	@printf '${BOLD}${RED}make: *** [setup-environment]${RESET}${EOL}'
	$(eval SRC_PATH := $(shell pwd))
	$(eval TMP_PATH := $(shell mktemp -d))
	$(eval GIT_PATH := $(shell go list -m))
	$(eval MOD_PATH := ${TMP_PATH}/src/${GIT_PATH})
	$(eval VER_DESC := $(shell git describe --tags))
	export GOPATH='${TMP_PATH}'
	export GO111MODULE='on'
	mkdir -p ${MOD_PATH}
	rmdir ${MOD_PATH}
	ln -Fs ${SRC_PATH} ${MOD_PATH}
.PHONY: setup-environment

test:
	@printf '${BOLD}${RED}make: *** [test]${RESET}${EOL}'
	ginkgo ./...
.PHONY: test

argocdproject/plugin: setup-environment
	@printf '${BOLD}${RED}make: *** [argocdproject/plugin]${RESET}${EOL}'
	cd ${MOD_PATH}                              && \
	go build                                       \
		-o 'argocdproject/plugin'                 \
		-a                                         \
		-installsuffix 'cgo'                       \
		-gcflags 'all=-trimpath "${TMP_PATH}/src"' \
		-v                                         \
		./argocdproject

clusterroles/plugin: setup-environment
	@printf '${BOLD}${RED}make: *** [clusterroles/plugin]${RESET}${EOL}'
	cd ${MOD_PATH}                              && \
	go build                                       \
		-o 'clusterroles/plugin'                   \
		-a                                         \
		-installsuffix 'cgo'                       \
		-gcflags 'all=-trimpath "${TMP_PATH}/src"' \
		-v                                         \
		./clusterroles

kustomizebuild/plugin: setup-environment
	@printf '${BOLD}${RED}make: *** [kustomizebuild/plugin]${RESET}${EOL}'
	cd ${MOD_PATH}                              && \
	go build                                       \
		-o 'kustomizebuild/plugin'                      \
		-a                                         \
		-installsuffix 'cgo'                       \
		-gcflags 'all=-trimpath "${TMP_PATH}/src"' \
		-v                                         \
		./kustomizebuild

namespace/plugin: setup-environment
	@printf '${BOLD}${RED}make: *** [namespace/plugin]${RESET}${EOL}'
	cd ${MOD_PATH}                              && \
	go build                                       \
		-o 'namespace/plugin'                      \
		-a                                         \
		-installsuffix 'cgo'                       \
		-gcflags 'all=-trimpath "${TMP_PATH}/src"' \
		-v                                         \
		./namespace

template/plugin: setup-environment
	@printf '${BOLD}${RED}make: *** [template/plugin]${RESET}${EOL}'
	cd ${MOD_PATH}                              && \
	go build                                       \
		-o 'template/plugin'                      \
		-a                                         \
		-installsuffix 'cgo'                       \
		-gcflags 'all=-trimpath "${TMP_PATH}/src"' \
		-v                                         \
		./template

unnamespaced/plugin: setup-environment
	@printf '${BOLD}${RED}make: *** [unnamespaced/plugin]${RESET}${EOL}'
	cd ${MOD_PATH}                              && \
	go build                                       \
		-o 'unnamespaced/plugin'                   \
		-a                                         \
		-installsuffix 'cgo'                       \
		-gcflags 'all=-trimpath "${TMP_PATH}/src"' \
		-v                                         \
		./unnamespaced

iac/charts:
	helm package iac/charts/charts/ci --destination iac/charts/charts
	helm package iac/charts/charts/cd --destination iac/charts/charts
	helm package iac/charts/charts/argocd --destination iac/charts/charts
.PHONY: iac/charts

iac/plugin: setup-environment iac/charts
	@printf '${BOLD}${RED}make: *** [iac/plugin]${RESET}${EOL}'
	cd ${MOD_PATH}/iac                          && \
	go build                                       \
		-o 'plugin'                                \
		-a                                         \
		-installsuffix 'cgo'                       \
		-gcflags 'all=-trimpath "${TMP_PATH}/src"' \
		-v                                         \
		.

build: argocdproject/plugin clusterroles/plugin kustomizebuild/plugin namespace/plugin template/plugin unnamespaced/plugin iac/plugin
.PHONY: build

install-argocdproject: argocdproject/plugin
	@printf '${BOLD}${RED}make: *** [install-argocdproject]${RESET}${EOL}'
	mkdir -p ${PLACEMENT}/argocdproject
	cp ./argocdproject/plugin ${PLACEMENT}/argocdproject/ArgoCDProject
.PHONY: install-argocdproject

install-clusterroles: clusterroles/plugin
	@printf '${BOLD}${RED}make: *** [install-clusterroles]${RESET}${EOL}'
	mkdir -p ${PLACEMENT}/clusterroles
	cp ./clusterroles/plugin ${PLACEMENT}/clusterroles/ClusterRoles
.PHONY: install-clusterroles

install-kustomizebuild: kustomizebuild/plugin
	@printf '${BOLD}${RED}make: *** [install-kustomizebuild]${RESET}${EOL}'
	mkdir -p ${PLACEMENT}/kustomizebuild
	cp ./kustomizebuild/plugin ${PLACEMENT}/kustomizebuild/KustomizeBuild
.PHONY: install-kustomizebuild

install-namespace: namespace/plugin
	@printf '${BOLD}${RED}make: *** [install-namespace]${RESET}${EOL}'
	mkdir -p ${PLACEMENT}/namespace
	cp ./namespace/plugin ${PLACEMENT}/namespace/Namespace
.PHONY: install-namespace

install-template: template/plugin
	@printf '${BOLD}${RED}make: *** [install-template]${RESET}${EOL}'
	mkdir -p ${PLACEMENT}/template
	cp ./template/plugin ${PLACEMENT}/template/Template
.PHONY: install-template

install-unnamespaced: unnamespaced/plugin
	@printf '${BOLD}${RED}make: *** [install-unnamespaced]${RESET}${EOL}'
	mkdir -p ${PLACEMENT}/unnamespaced
	cp ./unnamespaced/plugin ${PLACEMENT}/unnamespaced/Unnamespaced
.PHONY: install-unnamespaced

install-iac: iac/plugin
	@printf '${BOLD}${RED}make: *** [install-iac]${RESET}${EOL}'
	mkdir -p ${PLACEMENT}/iac
	cp ./iac/plugin ${PLACEMENT}/iac/iac
.PHONY: install-iac

install: install-argocdproject install-clusterroles install-kustomizebuild install-namespace install-template install-unnamespaced install-iac
.PHONY: install
