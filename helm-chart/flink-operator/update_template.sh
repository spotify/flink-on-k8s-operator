#!/bin/bash

manifests=$(mktemp /tmp/flink-operator-manifests.yaml)

function yqi() {
  yq -i "$1" "$manifests"
}

function modifyManifests() {
  deploymentSelector='select(.kind == "Deployment")'
  containersSelector="$deploymentSelector.spec.template.spec.containers"
  managerSelector="($containersSelector | .[] | select(.name == \"manager\"))"
  rbacProxySelector="($containersSelector | .[] | select(.name == \"kube-rbac-proxy\"))"

  yqi "$rbacProxySelector"'.image = "__RBAC_PROXY_IMAGE__"'
  yqi "$rbacProxySelector"'.imagePullPolicy = "__RBAC_PROXY_IMAGE_PULL_POLICY__"'
  yqi "del($rbacProxySelector.resources)"
  yqi "$rbacProxySelector |= sort_keys(.)"

  yqi "$managerSelector"'.args += "--watch-namespace=__WATCH_NAMESPACE__"'
  yqi "$managerSelector"'.resources.limits.cpu = "__LIMITS_CPU__"'
  yqi "$managerSelector"'.resources.limits.memory = "__LIMITS_MEMORY__"'
  yqi "$managerSelector"'.resources.requests.cpu = "__REQUESTS_CPU__"'
  yqi "$managerSelector"'.resources.requests.memory = "__REQUESTS_MEMORY__"'
  yqi "$managerSelector"'.image = "__MANAGER_IMAGE__"'
  yqi "$managerSelector"'.imagePullPolicy = "__MANAGER_IMAGE_PULL_POLICY__"'
  yqi "$managerSelector"' |= sort_keys(.)'

  yqi "$containersSelector |= sort_by(.name)"
  yqi "$managerSelector"'.name = "flink-operator"'
  yqi "$deploymentSelector"'.spec.template.metadata.annotations["kubectl.kubernetes.io/default-container"] = "flink-operator"'

  yqi "$deploymentSelector"'.spec.replicas = "__REPLICAS__"'
  yqi "$deploymentSelector"'.spec.template.spec.serviceAccountName = "__SERVICE_ACCOUNT__"'
  yqi '(select(.kind == "ClusterRoleBinding" or .kind == "RoleBinding").subjects[] | select(.kind == "ServiceAccount")).name = "__SERVICE_ACCOUNT__"'
}

function helmTemplating() {
  sed 's/__WATCH_NAMESPACE__/{{ .Values.watchNamespace.name }}/' |
  sed 's/__SERVICE_ACCOUNT__/{{ template "flink-operator.serviceAccountName" . }}/' |
  sed 's/__NAMESPACE__/{{ .Values.flinkOperatorNamespace.name }}/g' |
  sed 's/__LIMITS_CPU__/{{ .Values.resources.limits.cpu }}/' |
  sed 's/__LIMITS_MEMORY__/{{ .Values.resources.limits.memory }}/' |
  sed 's/__REQUESTS_CPU__/{{ .Values.resources.requests.cpu }}/' |
  sed 's/__REQUESTS_MEMORY__/{{ .Values.resources.requests.memory }}/' |
  sed 's/__MANAGER_IMAGE_PULL_POLICY__/{{ .Values.operatorImage.pullPolicy }}/' |
  sed 's/__MANAGER_IMAGE__/{{ .Values.operatorImage.name }}/' |
  sed 's/__RBAC_PROXY_IMAGE__/{{ .Values.rbacProxyImage.name }}/' |
  sed 's/__RBAC_PROXY_IMAGE_PULL_POLICY__/{{ .Values.rbacProxyImage.pullPolicy }}/' |
  sed 's/__REPLICAS__/{{ .Values.replicas }}/g'
}

function separateManifests() {
  yq 'select(.apiVersion == "apiextensions.k8s.io/v1")' "$manifests" | \
  helmTemplating > templates/flink-cluster-crd.yaml

  yq 'select(.apiVersion == "rbac.authorization.k8s.io/v1")' "$manifests" | \
  (echo "{{- if .Values.rbac.create }}" && cat && echo "{{- end }}") | \
  helmTemplating > templates/rbac.yaml

  yq 'select(.apiVersion == "cert-manager.io/v1" or .apiVersion == "admissionregistration.k8s.io/v1")' "$manifests" | \
  helmTemplating > templates/webhook.yaml

  read -r -d '' operatorSelector << EOM
select(true
and .apiVersion != "apiextensions.k8s.io/v1"
and .apiVersion != "rbac.authorization.k8s.io/v1"
and .apiVersion != "cert-manager.io/v1"
and .apiVersion != "admissionregistration.k8s.io/v1"
and .kind != "Namespace"
and .kind != "ServiceAccount"
)
EOM
  yq "$operatorSelector" "$manifests" | \
  helmTemplating > templates/flink-operator.yaml
}

function main() {
  sourceKustomization="../../config/default/kustomization.yaml"
  yq -i '.namespace = "__NAMESPACE__"' "$sourceKustomization"
  kubectl kustomize "$(dirname $sourceKustomization)" > "$manifests"
  modifyManifests
  separateManifests
  rm "$manifests"
  git checkout "$sourceKustomization"
}

main