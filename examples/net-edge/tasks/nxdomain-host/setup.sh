#!/usr/bin/env bash
set -euo pipefail

if ! command -v oc >/dev/null 2>&1; then
  echo "oc CLI is required to stage the scenario" >&2
  exit 1
fi

NAMESPACE="${NAMESPACE:-netedge-scenario2}"
APP_NAME="${APP_NAME:-hello}"
APP_LABEL="${APP_LABEL:-hello}"
IMAGE="${IMAGE:-quay.io/openshift/origin-hello-openshift:latest}"
PORT="${PORT:-8080}"
BREAK_HOST_TEMPLATE="${BREAK_HOST:-broken-%s.invalid}"
ANNOTATION_KEY="netedge-tools-original-host"

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

route_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.host}' 2>/dev/null || true)"
if [[ -z "${route_host}" ]]; then
  echo "Route host is not admitted yet; setup cannot continue" >&2
  exit 1
fi
echo "Healthy baseline ready with route host ${route_host}"

echo "Annotating route with original host"
oc -n "${NAMESPACE}" annotate route "${APP_NAME}" "${ANNOTATION_KEY}=${route_host}" --overwrite

random_suffix="$(python3 - <<'PY'
import secrets, string
alphabet = string.ascii_lowercase + string.digits
print(''.join(secrets.choice(alphabet) for _ in range(6)))
PY
)"
random_suffix="${random_suffix:-nxdomn}"
broken_host=$(printf "${BREAK_HOST_TEMPLATE}" "${random_suffix}")

echo "Patching route host to NXDOMAIN value ${broken_host}"
oc -n "${NAMESPACE}" patch route "${APP_NAME}" --type=merge -p "{\"spec\":{\"host\":\"${broken_host}\"}}"

echo "Verifying route host now set to ${broken_host}"
current_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.host}' 2>/dev/null || true)"
if [[ "${current_host}" != "${broken_host}" ]]; then
  echo "Route host patch failed (expected ${broken_host}, got ${current_host:-<none>})" >&2
  exit 1
fi

echo "Scenario 2 staging complete. Route host now points at NXDOMAIN value ${broken_host}."
