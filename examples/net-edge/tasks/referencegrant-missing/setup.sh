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

if [[ -z "${GATEWAY_CLASS}" ]]; then
  GATEWAY_CLASS="$(oc get gatewayclass -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
  if [[ -z "${GATEWAY_CLASS}" ]]; then
    echo "No GatewayClass available; set GATEWAY_CLASS to a valid value before running" >&2
    exit 1
  fi
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
oc delete referencegrant allow-${APP_NAME} -n "${BACKEND_NAMESPACE}" --ignore-not-found

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
for _ in $(seq 1 24); do
  resolved="$(oc -n "${GATEWAY_NAMESPACE}" get httproute "${APP_NAME}" -o json | jq -r '.status.parents[]?.conditions[]? | select(.type=="ResolvedRefs") | .status' 2>/dev/null || true)"
  if [[ "${resolved}" == *"False"* ]]; then
    break
  fi
  sleep 5
done

resolved="$(oc -n "${GATEWAY_NAMESPACE}" get httproute "${APP_NAME}" -o json | jq -r '.status.parents[]?.conditions[]? | select(.type=="ResolvedRefs") | .status' 2>/dev/null || true)"
if [[ "${resolved}" != *"False"* ]]; then
  echo "HTTPRoute did not surface ResolvedRefs=False; ensure the gateway controller is reporting status" >&2
  exit 1
fi

if oc -n "${BACKEND_NAMESPACE}" get referencegrant allow-"${APP_NAME}" >/dev/null 2>&1; then
  echo "ReferenceGrant already present; scenario requires it to be missing" >&2
  exit 1
fi

echo "Scenario 6 staging complete. HTTPRoute references backend service across namespaces without a ReferenceGrant."
