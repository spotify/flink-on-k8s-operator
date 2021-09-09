
# Image URL to use all building/pushing image targets
IMG ?= gcr.io/esquilo/regadas/flink-operator:snapshot
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:maxDescLen=0,trivialVersions=true,preserveUnknownFields=false,generateEmbeddedObjectMeta=true"
# The Kubernetes namespace to limit watching.
WATCH_NAMESPACE ?=

#################### Local build and test ####################

all: build

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./api/v1beta1/..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/v1beta1/..."

tidy: ## Run go mod tidy
	go mod tidy

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet tidy kustomize ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	$(KUSTOMIZE) build config/crd > config/crd/bases/patched_crd.yaml
	mv config/crd/bases/patched_crd.yaml config/crd/bases/flinkoperator.k8s.io_flinkclusters.yaml
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build

build: generate fmt vet tidy ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet tidy ## Run a controller from your host against the configured Kubernetes cluster in ~/.kube/config
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} --label git-commit=$(shell git rev-parse HEAD) .

docker-push: docker-build ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: install ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# # Build kustomize overlay for deploy/undeploy.
# build-overlay:
# 	rm -rf config/deploy && cp -rf config/default config/deploy && cd config/deploy \
# 			&& kustomize edit set nameprefix $(RESOURCE_PREFIX) \
# 			&& kustomize edit set namespace $(FLINK_OPERATOR_NAMESPACE)
# ifneq ($(WATCH_NAMESPACE),)
# 	cd config/deploy \
# 			&& sed -E -i.bak  "s/(\-\-watch\-namespace\=)/\1$(WATCH_NAMESPACE)/" manager_auth_proxy_patch.yaml \
# 			&& kustomize edit add patch webhook_namespace_selector_patch.yaml \
# 			|| true
# endif
# 	sed -E -i.bak "s/resources:/bases:/" config/deploy/kustomization.yaml
# 	rm config/deploy/*.bak

# Generate deploy template.
# template: build-overlay
# 	kubectl kustomize config/deploy \
# 			| sed -e "s/$(RESOURCE_PREFIX)system/$(FLINK_OPERATOR_NAMESPACE)/g"

# Deploy the operator in the configured Kubernetes cluster in ~/.kube/config
# old-deploy: install webhook-cert config/default/manager_image_patch.yaml build-overlay
# 	sed -e 's#image: .*#image: '"$(IMG)"'#' ./config/deploy/manager_image_patch.template >./config/deploy/manager_image_patch.yaml
# 	@echo "Getting webhook server certificate"
# 	$(eval CA_BUNDLE := $(shell kubectl get secrets/webhook-server-cert -n $(FLINK_OPERATOR_NAMESPACE) -o jsonpath="{.data.tls\.crt}"))
# 	kubectl kustomize config/deploy \
# 			| sed -e "s/$(RESOURCE_PREFIX)system/$(FLINK_OPERATOR_NAMESPACE)/g" \
# 			| sed -e "s/Cg==/$(CA_BUNDLE)/g" \
# 			| kubectl apply -f -
# ifneq ($(WATCH_NAMESPACE),)
#     # Set the label on watch-target namespace to support webhook namespaceSelector.
# 	kubectl label ns $(WATCH_NAMESPACE) flink-operator-namespace=$(FLINK_OPERATOR_NAMESPACE)
# endif
# 	@printf "$(GREEN)Flink Operator deployed, image=$(IMG), operator_namespace=$(FLINK_OPERATOR_NAMESPACE), watch_namespace=$(WATCH_NAMESPACE)$(RESET)\n"


# undeploy-controller: build-overlay
# 	kubectl kustomize config/deploy \
# 			| sed -e "s/$(RESOURCE_PREFIX)system/$(FLINK_OPERATOR_NAMESPACE)/g" \
# 			| kubectl delete -f - \
# 			|| true
# ifneq ($(WATCH_NAMESPACE),)
#     # Remove the label, which is set when operator is deployed to support webhook namespaceSelector
# 	kubectl label ns $(WATCH_NAMESPACE) flink-operator-namespace-
# endif

# Deploy the sample Flink clusters in the Kubernetes cluster
samples:
	kubectl apply -f config/samples/

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.2)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
