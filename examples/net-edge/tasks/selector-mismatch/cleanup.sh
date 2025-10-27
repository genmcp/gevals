#!/usr/bin/env bash
set -euo pipefail

if ! command -v oc >/dev/null 2>&1; then
  echo "oc CLI is required for cleanup" >&2
  exit 1
fi

NAMESPACE="${NAMESPACE:-netedge-scenario1}"
APP_NAME="${APP_NAME:-hello}"
DELETE_NAMESPACE="${DELETE_NAMESPACE:-false}"

echo "Removing workload resources from ${NAMESPACE}"
oc delete route "${APP_NAME}" -n "${NAMESPACE}" --ignore-not-found
oc delete svc "${APP_NAME}" -n "${NAMESPACE}" --ignore-not-found
oc delete deploy "${APP_NAME}" -n "${NAMESPACE}" --ignore-not-found

if [[ "${DELETE_NAMESPACE}" == "true" ]]; then
  echo "Deleting namespace ${NAMESPACE}"
  oc delete namespace "${NAMESPACE}" --ignore-not-found
fi
