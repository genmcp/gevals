#!/usr/bin/env bash
set -euo pipefail

if ! command -v oc >/dev/null 2>&1; then
  echo "oc CLI is required to stage the scenario" >&2
  exit 1
fi

NAMESPACE="${NAMESPACE:-netedge-scenario4}"
APP_NAME="${APP_NAME:-hello}"
APP_LABEL="${APP_LABEL:-hello}"
IMAGE="${IMAGE:-quay.io/openshift/origin-hello-openshift:latest}"
PORT="${PORT:-8080}"

echo "Preparing namespace ${NAMESPACE}"
if ! oc get namespace "${NAMESPACE}" >/dev/null 2>&1; then
  oc create namespace "${NAMESPACE}"
else
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

echo "Patching route to reencrypt termination"
oc -n "${NAMESPACE}" patch route "${APP_NAME}" --type=merge -p '{"spec":{"tls":{"termination":"reencrypt","certificate":"dummy-cert","key":"dummy-key","destinationCACertificate":"dummy-ca","insecureEdgeTerminationPolicy":"Redirect"}}}'

echo "Scenario 4 staging complete. Route now enforces reencrypt while backend serves plain HTTP."
