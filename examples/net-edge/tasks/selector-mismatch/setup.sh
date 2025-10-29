#!/usr/bin/env bash
set -euo pipefail

if ! command -v oc >/dev/null 2>&1; then
  echo "oc CLI is required to stage the scenario" >&2
  exit 1
fi

NAMESPACE="${NAMESPACE:-netedge-scenario1}"
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

healthy_ip="$(oc -n "${NAMESPACE}" get endpoints "${APP_NAME}" -o jsonpath='{.subsets[0].addresses[0].ip}' 2>/dev/null || true)"
if [[ -z "${healthy_ip}" ]]; then
  echo "Endpoints failed to populate after rollout" >&2
  exit 1
fi

route_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.host}' 2>/dev/null || true)"
echo "Healthy baseline ready: selector app=${APP_LABEL}, endpoint IP ${healthy_ip}, route host ${route_host:-<pending>}"

echo "Breaking service selector to create empty endpoints"
oc -n "${NAMESPACE}" patch svc "${APP_NAME}" --type=merge -p '{"spec":{"selector":{"app":"broken-mismatch"}}}'
broken_selector="$(oc -n "${NAMESPACE}" get svc "${APP_NAME}" -o jsonpath='{.spec.selector.app}' 2>/dev/null || true)"
echo "Service selector now set to '${broken_selector}' (expected broken-mismatch)"

echo "Waiting for endpoints to become empty"
for _ in $(seq 1 24); do
  subsets="$(oc -n "${NAMESPACE}" get endpoints "${APP_NAME}" -o jsonpath='{.subsets}' 2>/dev/null || true)"
  if [[ -z "${subsets}" || "${subsets}" == "[]" ]]; then
    echo "Confirmed endpoints empty after selector mismatch: ${subsets:-[]}"
    exit 0
  fi
  sleep 5
done

echo "Endpoints still populated after selector break; failing setup" >&2
exit 1
