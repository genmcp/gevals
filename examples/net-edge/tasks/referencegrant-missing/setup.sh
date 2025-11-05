#!/usr/bin/env bash
set -euo pipefail

for bin in oc jq; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "${bin} CLI is required to stage the scenario" >&2
    exit 1
  fi
done

GATEWAY_NAMESPACE="${GATEWAY_NAMESPACE:-netedge-scenario6}"
BACKEND_NAMESPACE="${BACKEND_NAMESPACE:-netedge-scenario6-backend}"
APP_NAME="${APP_NAME:-hello}"
APP_LABEL="${APP_LABEL:-hello}"
IMAGE="${IMAGE:-quay.io/openshift/origin-hello-openshift:latest}"
PORT="${PORT:-8080}"
GATEWAY_CLASS="${GATEWAY_CLASS:-}"
HOSTNAME="${HOSTNAME:-hello-scenario6.netedge.example}"
TEMP_GATEWAY_CLASS="${TEMP_GATEWAY_CLASS:-netedge-temp-gatewayclass}"
TEMP_GATEWAY_LABEL_KEY="${TEMP_GATEWAY_LABEL_KEY:-netedge.openshift.io/scenario}"
TEMP_GATEWAY_LABEL_VALUE="${TEMP_GATEWAY_LABEL_VALUE:-referencegrant-missing}"
GATEWAY_CONTROLLER_NAME="${GATEWAY_CONTROLLER_NAME:-openshift.io/gateway-controller/v1}"

tmp_err="$(mktemp)"
gatewayclass_names="$(oc get gatewayclass -o jsonpath='{.items[*].metadata.name}' 2>"${tmp_err}" || true)"
cmd_status=$?
err_msg="$(<"${tmp_err}")"
rm -f "${tmp_err}"

if [[ ${cmd_status} -ne 0 ]]; then
  if [[ "${err_msg}" == *"the server doesn't have a resource type \"gatewayclass\""* ]]; then
    if [[ -z "${GATEWAY_CLASS}" ]]; then
      echo "Gateway API (GatewayClass) not available on this cluster; set GATEWAY_CLASS to an existing class or enable the feature before running" >&2
      exit 1
    fi
  else
    if [[ -n "${err_msg}" ]]; then
      echo "${err_msg}" >&2
    fi
    exit 1
  fi
fi

if [[ -z "${GATEWAY_CLASS}" && -n "${gatewayclass_names}" ]]; then
  GATEWAY_CLASS="${gatewayclass_names%% *}"
fi

if [[ -z "${GATEWAY_CLASS}" ]]; then
  include_parameters_ref="false"
  if oc -n openshift-ingress-operator get ingresscontroller default >/dev/null 2>&1; then
    include_parameters_ref="true"
  fi

  create_gateway_class_manifest() {
    local api_version="$1"
    local with_params="$2"
    cat <<YAML
apiVersion: ${api_version}
kind: GatewayClass
metadata:
  name: ${TEMP_GATEWAY_CLASS}
  labels:
    ${TEMP_GATEWAY_LABEL_KEY}: ${TEMP_GATEWAY_LABEL_VALUE}
spec:
  controllerName: ${GATEWAY_CONTROLLER_NAME}
YAML
    if [[ "${with_params}" == "true" ]]; then
      cat <<YAML
  parametersRef:
    group: operator.openshift.io
    kind: IngressController
    name: default
    namespace: openshift-ingress-operator
YAML
    fi
  }

  applied="false"
  last_error=""
  for api_version in "gateway.networking.k8s.io/v1" "gateway.networking.k8s.io/v1beta1"; do
    if [[ "${include_parameters_ref}" == "true" ]]; then
      if output="$(create_gateway_class_manifest "${api_version}" "true" | oc apply -f - 2>&1)"; then
        applied="true"
        break
      else
        last_error="${output}"
      fi
    fi
    if output="$(create_gateway_class_manifest "${api_version}" "false" | oc apply -f - 2>&1)"; then
      applied="true"
      break
    else
      last_error="${output}"
    fi
  done

  if [[ "${applied}" != "true" ]]; then
    if [[ -n "${last_error}" ]]; then
      echo "${last_error}" >&2
    fi
    echo "Failed to create temporary GatewayClass ${TEMP_GATEWAY_CLASS}; set GATEWAY_CLASS to a valid value before running" >&2
    exit 1
  fi

  GATEWAY_CLASS="${TEMP_GATEWAY_CLASS}"
fi

if ! oc get gatewayclass "${GATEWAY_CLASS}" >/dev/null 2>&1; then
  echo "GatewayClass ${GATEWAY_CLASS} does not exist" >&2
  exit 1
fi

echo "Preparing namespaces ${GATEWAY_NAMESPACE} and ${BACKEND_NAMESPACE}"
oc get namespace "${GATEWAY_NAMESPACE}" >/dev/null 2>&1 || oc create namespace "${GATEWAY_NAMESPACE}"
oc get namespace "${BACKEND_NAMESPACE}" >/dev/null 2>&1 || oc create namespace "${BACKEND_NAMESPACE}"

echo "Resetting resources in ${GATEWAY_NAMESPACE}"
oc delete httproute,gateway "${APP_NAME}" -n "${GATEWAY_NAMESPACE}" --ignore-not-found

echo "Resetting resources in ${BACKEND_NAMESPACE}"
oc delete deploy,svc "${APP_NAME}" -n "${BACKEND_NAMESPACE}" --ignore-not-found
oc delete referencegrant -n "${BACKEND_NAMESPACE}" --all --ignore-not-found

echo "Deploying backend workload in ${BACKEND_NAMESPACE}"
cat <<YAML | oc -n "${BACKEND_NAMESPACE}" apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${APP_NAME}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${APP_LABEL}
  template:
    metadata:
      labels:
        app: ${APP_LABEL}
    spec:
      containers:
      - name: ${APP_NAME}
        image: ${IMAGE}
        ports:
        - containerPort: ${PORT}
---
apiVersion: v1
kind: Service
metadata:
  name: ${APP_NAME}
spec:
  selector:
    app: ${APP_LABEL}
  ports:
  - name: http
    port: ${PORT}
    targetPort: ${PORT}
YAML

echo "Waiting for backend rollout"
oc -n "${BACKEND_NAMESPACE}" rollout status deploy/"${APP_NAME}" --timeout=120s

echo "Deploying Gateway and HTTPRoute without ReferenceGrant"
cat <<YAML | oc -n "${GATEWAY_NAMESPACE}" apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: ${APP_NAME}
spec:
  gatewayClassName: ${GATEWAY_CLASS}
  listeners:
  - name: http
    protocol: HTTP
    port: 80
    hostname: ${HOSTNAME}
    allowedRoutes:
      namespaces:
        from: Same
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: ${APP_NAME}
spec:
  parentRefs:
  - name: ${APP_NAME}
    namespace: ${GATEWAY_NAMESPACE}
  rules:
  - backendRefs:
    - name: ${APP_NAME}
      namespace: ${BACKEND_NAMESPACE}
      port: ${PORT}
YAML

echo "Waiting for HTTPRoute status to indicate unresolved references"
is_unresolved="false"
for _ in $(seq 1 36); do
  status_json="$(oc -n "${GATEWAY_NAMESPACE}" get httproute "${APP_NAME}" -o json 2>/dev/null || true)"
  if [[ -n "${status_json}" ]]; then
    is_unresolved="$(printf '%s' "${status_json}" | jq -r '
      [
        .status.parents[]?.conditions[]?
        | select(.type=="ResolvedRefs")
        | ((.status == "False")
           or (.reason == "RefNotPermitted")
           or ((.message // "") | contains("missing a ReferenceGrant")))
      ]
      | any
      ' 2>/dev/null || echo false)"
    if [[ "${is_unresolved}" == "true" ]]; then
      break
    fi
  fi
  sleep 5
done

if [[ "${is_unresolved}" != "true" ]]; then
  status_json="$(oc -n "${GATEWAY_NAMESPACE}" get httproute "${APP_NAME}" -o json 2>/dev/null || true)"
  resolved_status="$(printf '%s' "${status_json}" | jq -r '.status.parents[]?.conditions[]? | select(.type=="ResolvedRefs") | .status' 2>/dev/null || echo '')"
  resolved_reason="$(printf '%s' "${status_json}" | jq -r '.status.parents[]?.conditions[]? | select(.type=="ResolvedRefs") | .reason' 2>/dev/null || echo '')"
  echo "HTTPRoute did not surface ResolvedRefs=False (status='${resolved_status}', reason='${resolved_reason}'); ensure the gateway controller is reporting status" >&2
  exit 1
fi

if oc -n "${BACKEND_NAMESPACE}" get referencegrant allow-"${APP_NAME}" >/dev/null 2>&1; then
  echo "ReferenceGrant already present; scenario requires it to be missing" >&2
  exit 1
fi

echo "Scenario 6 staging complete. HTTPRoute references backend service across namespaces without a ReferenceGrant."
