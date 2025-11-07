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
TEMP_GATEWAY_LABEL_KEY="${TEMP_GATEWAY_LABEL_KEY:-netedge.openshift.io/scenario}"
TEMP_GATEWAY_LABEL_VALUE="${TEMP_GATEWAY_LABEL_VALUE:-referencegrant-missing}"

echo "Removing Gateway resources from ${GATEWAY_NAMESPACE}"
oc delete httproute "${APP_NAME}" -n "${GATEWAY_NAMESPACE}" --ignore-not-found
oc delete gateway "${APP_NAME}" -n "${GATEWAY_NAMESPACE}" --ignore-not-found

echo "Removing backend workload from ${BACKEND_NAMESPACE}"
oc delete referencegrant -n "${BACKEND_NAMESPACE}" --all --ignore-not-found
oc delete svc "${APP_NAME}" -n "${BACKEND_NAMESPACE}" --ignore-not-found
oc delete deploy "${APP_NAME}" -n "${BACKEND_NAMESPACE}" --ignore-not-found

if gateway_api_resources="$(oc api-resources --api-group=gateway.networking.k8s.io 2>/dev/null | awk 'NR>1 {print $1}' | tr '\n' ' ')"; then
  if [[ " ${gateway_api_resources} " == *" GatewayClass "* ]]; then
    oc delete gatewayclass -l "${TEMP_GATEWAY_LABEL_KEY}=${TEMP_GATEWAY_LABEL_VALUE}" --ignore-not-found
  fi
fi

if [[ "${DELETE_NAMESPACE}" == "true" ]]; then
  oc delete namespace "${GATEWAY_NAMESPACE}" --ignore-not-found
  oc delete namespace "${BACKEND_NAMESPACE}" --ignore-not-found
fi
