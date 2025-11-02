#!/usr/bin/env bash
set -euo pipefail

if ! command -v oc >/dev/null 2>&1; then
  echo "oc CLI is required for cleanup" >&2
  exit 1
fi

GATEWAY_NAMESPACE="${GATEWAY_NAMESPACE:-netedge-scenario6}"
BACKEND_NAMESPACE="${BACKEND_NAMESPACE:-netedge-scenario6-backend}"
APP_NAME="${APP_NAME:-hello}"
DELETE_NAMESPACE="${DELETE_NAMESPACE:-false}"

echo "Removing Gateway resources from ${GATEWAY_NAMESPACE}"
oc delete httproute "${APP_NAME}" -n "${GATEWAY_NAMESPACE}" --ignore-not-found
oc delete gateway "${APP_NAME}" -n "${GATEWAY_NAMESPACE}" --ignore-not-found

echo "Removing backend workload from ${BACKEND_NAMESPACE}"
oc delete referencegrant allow-"${APP_NAME}" -n "${BACKEND_NAMESPACE}" --ignore-not-found
oc delete svc "${APP_NAME}" -n "${BACKEND_NAMESPACE}" --ignore-not-found
oc delete deploy "${APP_NAME}" -n "${BACKEND_NAMESPACE}" --ignore-not-found

if [[ "${DELETE_NAMESPACE}" == "true" ]]; then
  oc delete namespace "${GATEWAY_NAMESPACE}" --ignore-not-found
  oc delete namespace "${BACKEND_NAMESPACE}" --ignore-not-found
fi
