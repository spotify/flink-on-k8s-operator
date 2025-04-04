SHELL := /bin/bash
VERSION ?= latest
# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/spotify/flink-operator:$(VERSION)
# Image platforms to support.
PLATFORMS ?= linux/amd64,linux/arm64
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:maxDescLen=0,generateEmbeddedObjectMeta=true"
# The Kubernetes namespace in which the operator will be deployed.
FLINK_OPERATOR_NAMESPACE ?= flink-operator-system
# Prefix for Kubernetes resource names. When deploying multiple operators, make sure that the names of cluster-scoped resources are not duplicated.
RESOURCE_PREFIX ?= flink-operator-
# The Kubernetes namespace to limit watching.
WATCH_NAMESPACE ?=

KUSTOMIZE_VERSION=v4.5.7
CONTROLLER_GEN_VERSION=v0.16.5
# Env test configuration
ENVTEST_K8S_VERSION=1.26.0

LOCALBIN=$(shell pwd)/bin

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./apis/flinkcluster/v1beta1/..." output:crd:artifacts:config=config/crd/bases
	# remove status field as they interfer with ArgoCD and Google config-sync
	# https://github.com/kubernetes-sigs/controller-tools/issues/456
	yq -i e 'del(.status)' config/crd/bases/flinkoperator.k8s.io_flinkclusters.yaml

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/flinkcluster/v1beta1/..."

generate-crd-docs: crd-ref-docs ## Generate CRD documentation to docs/crd.md
	$(CRD_REF_DOCS) --source-path=./apis/flinkcluster/v1beta1 --config=docs/config.yaml --renderer=markdown --output-path=docs/crd.md

tidy: ## Run go mod tidy
	go mod tidy

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet tidy kustomize envtest ## Run tests.
	rm -rf config/test && mkdir -p config/test/crd
	$(KUSTOMIZE) build config/crd > config/test/crd/flinkoperator.k8s.io_flinkclusters.yaml
	KUBEBUILDER_ASSETS=$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path) go test ./... -coverprofile cover.out
##@ Build

build: generate fmt vet tidy ## Build manager binary.
	go build -o bin/manager main.go

build-overlay: manifests kustomize ## Build overlay for deployment.
	rm -rf config/deploy && cp -rf config/default config/deploy && cd config/deploy \
	    && $(KUSTOMIZE) edit set image controller="${IMG}" \
		&& $(KUSTOMIZE) edit set nameprefix $(RESOURCE_PREFIX) \
		&& $(KUSTOMIZE) edit set namespace $(FLINK_OPERATOR_NAMESPACE)
ifneq ($(WATCH_NAMESPACE),)
	cd config/deploy \
			&& sed -E -i.bak  "s/(\-\-watch\-namespace\=)/\1$(WATCH_NAMESPACE)/" manager_auth_proxy_patch.yaml \
			&& kustomize edit add patch --path mutation_webhook_namespace_selector_patch.yaml \
			&& kustomize edit add patch --path validation_webhook_namespace_selector_patch.yaml \
			&& 	rm config/deploy/*.bak || true
endif

build-overlay-sharded: manifests kustomize ## Build overlay for deployment.
	rm -rf config/deploy-sharded && cp -rf config/default-sharded config/deploy-sharded && cd config/deploy-sharded \
	    && $(KUSTOMIZE) edit set image controller="${IMG}" \
		&& $(KUSTOMIZE) edit set nameprefix $(RESOURCE_PREFIX) \
		&& $(KUSTOMIZE) edit set namespace $(FLINK_OPERATOR_NAMESPACE)
ifneq ($(WATCH_NAMESPACE),)
	cd config/deploy-sharded \
			&& sed -E -i.bak  "s/(\-\-watch\-namespace\=)/\1$(WATCH_NAMESPACE)/" manager_auth_proxy_patch.yaml \
			&& kustomize edit add patch --path mutation_webhook_namespace_selector_patch.yaml \
			&& kustomize edit add patch --path validation_webhook_namespace_selector_patch.yaml \
			&& 	rm config/deploy-sharded/*.bak || true
endif

run: manifests generate fmt vet tidy ## Run a controller from your host against the configured Kubernetes cluster in ~/.kube/config
	go run ./main.go

.PHONY: docker-create-builder ## Create makes a new builder instance.
docker-create-builder:
	docker buildx create --use

docker-build: test docker-create-builder ## Build docker image with the manager.
	docker buildx build --platform ${PLATFORMS} -t ${IMG} --label git-commit=$(shell git rev-parse HEAD) .

docker-push: test docker-create-builder ## Push docker image with the manager.
	docker buildx build --push --platform ${PLATFORMS} -t ${IMG} --label git-commit=$(shell git rev-parse HEAD) .

release-manifests: build-overlay build-overlay-sharded ## Build manifests for release.
	$(KUSTOMIZE) build config/deploy > config/deploy/flink-operator.yaml \
	&& $(KUSTOMIZE) build config/deploy-sharded > config/deploy/flink-operator-sharded.yaml

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: install build-overlay ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/deploy | kubectl apply -f -

deploy-sharded: install build-overlay-sharded
	$(KUSTOMIZE) build config/deploy-sharded | kubectl apply -f -

ifneq ($(WATCH_NAMESPACE),)
    # Set the label on watch-target namespace to support webhook namespaceSelector.
	kubectl label ns $(WATCH_NAMESPACE) flink-operator-namespace=$(FLINK_OPERATOR_NAMESPACE)
endif

undeploy: build-overlay ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/deploy | kubectl delete -f -

undeploy-sharded: build-overlay-sharded
	$(KUSTOMIZE) build config/deploy-sharded | kubectl delete -f -

ifneq ($(WATCH_NAMESPACE),)
    # Remove the label, which is set when operator is deployed to support webhook namespaceSelector
	kubectl label ns $(WATCH_NAMESPACE) flink-operator-namespace-
endif

# Deploy the sample Flink clusters in the Kubernetes cluster
samples:
	kubectl apply -f config/samples/

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION))

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@$(KUSTOMIZE_VERSION))

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

CRD_REF_DOCS = $(shell pwd)/bin/crd-ref-docs
crd-ref-docs:
	$(call go-get-tool,$(CRD_REF_DOCS),github.com/elastic/crd-ref-docs@master)

# go-get-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
