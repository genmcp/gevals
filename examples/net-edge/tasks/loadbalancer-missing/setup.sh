#!/usr/bin/env bash
set -euo pipefail

for bin in oc jq; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "${bin} CLI is required to stage the scenario" >&2
    exit 1
  fi
done

NAMESPACE="${NAMESPACE:-netedge-scenario5}"
APP_NAME="${APP_NAME:-hello}"
APP_LABEL="${APP_LABEL:-hello}"
IMAGE="${IMAGE:-quay.io/openshift/origin-hello-openshift:latest}"
PORT="${PORT:-8080}"
LOAD_BALANCER_CLASS="${LOAD_BALANCER_CLASS:-example.com/unsupported}"

echo "Preparing namespace ${NAMESPACE}"
if ! oc get namespace "${NAMESPACE}" >/dev/null 2>&1; then
  oc create namespace "${NAMESPACE}"
else
  oc delete route,svc,deploy "${APP_NAME}" -n "${NAMESPACE}" --ignore-not-found
fi

echo "Deploying workload and LoadBalancer service"
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
  type: LoadBalancer
  loadBalancerClass: ${LOAD_BALANCER_CLASS}
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

service_json="$(oc -n "${NAMESPACE}" get svc "${APP_NAME}" -o json)"
external_ip="$(printf '%s' "${service_json}" | jq -r '.status.loadBalancer.ingress[0].ip // empty')"
if [[ -n "${external_ip}" ]]; then
  echo "Unexpected external IP ${external_ip} provisioned; scenario requires pending load balancer" >&2
  exit 1
fi

echo "Scenario 5 staging complete. Service ${APP_NAME} remains without an external IP due to unsupported LoadBalancer class ${LOAD_BALANCER_CLASS}."
