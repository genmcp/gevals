#!/usr/bin/env bash
set -euo pipefail

if ! command -v oc >/dev/null 2>&1; then
  echo "oc CLI is required to stage the scenario" >&2
  exit 1
fi

NAMESPACE="${NAMESPACE:-netedge-scenario3}"
APP_NAME="${APP_NAME:-hello}"
APP_LABEL="${APP_LABEL:-hello}"
IMAGE="${IMAGE:-quay.io/openshift/origin-hello-openshift:latest}"
PORT="${PORT:-8080}"
NP_NAME="${NP_NAME:-deny-router}"

echo "Preparing namespace ${NAMESPACE}"
if ! oc get namespace "${NAMESPACE}" >/dev/null 2>&1; then
  oc create namespace "${NAMESPACE}"
else
  oc delete networkpolicy "${NP_NAME}" -n "${NAMESPACE}" --ignore-not-found
  oc delete route,svc,deploy "${APP_NAME}" -n "${NAMESPACE}" --ignore-not-found
fi

echo "Deploying baseline workload"
cat <<YAML | oc -n "${NAMESPACE}" apply -f -
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
---
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: ${APP_NAME}
spec:
  to:
    kind: Service
    name: ${APP_NAME}
  port:
    targetPort: http
YAML

echo "Waiting for deployment rollout"
oc -n "${NAMESPACE}" rollout status deploy/"${APP_NAME}" --timeout=120s

echo "Waiting for endpoints to populate"
oc -n "${NAMESPACE}" wait --for=jsonpath='{.subsets[0].addresses[0].ip}' endpoints/"${APP_NAME}" --timeout=120s

route_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.host}' 2>/dev/null || true)"
if [[ -z "${route_host}" ]]; then
  echo "Route host is not admitted yet; setup cannot continue" >&2
  exit 1
fi
echo "Healthy baseline ready with route host ${route_host}"

echo "Applying default-deny NetworkPolicy ${NP_NAME}"
cat <<YAML | oc -n "${NAMESPACE}" apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ${NP_NAME}
spec:
  podSelector: {}
  policyTypes:
    - Ingress
  ingress: []
YAML

echo "Scenario 3 staging complete. Requests to ${route_host} should now fail due to the deny-all NetworkPolicy."
