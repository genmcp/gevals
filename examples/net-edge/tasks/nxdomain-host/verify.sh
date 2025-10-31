#!/usr/bin/env bash
set -euo pipefail

if ! command -v oc >/dev/null 2>&1; then
  echo "oc CLI is required for verification" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for verification" >&2
  exit 1
fi

NAMESPACE="${NAMESPACE:-netedge-scenario2}"
APP_NAME="${APP_NAME:-hello}"
ANNOTATION_KEY="netedge-tools-original-host"

expected_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath="{.metadata.annotations['${ANNOTATION_KEY}']}" 2>/dev/null || true)"
if [[ -z "${expected_host}" ]]; then
  echo "route ${NAMESPACE}/${APP_NAME} is missing ${ANNOTATION_KEY} annotation" >&2
  exit 1
fi

current_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.host}' 2>/dev/null || true)"
if [[ -z "${current_host}" ]]; then
  echo "route ${NAMESPACE}/${APP_NAME} has no admitted host" >&2
  exit 1
fi

if [[ "${current_host}" != "${expected_host}" ]]; then
  echo "route host still incorrect: current '${current_host}', expected '${expected_host}'" >&2
  exit 1
fi

echo "Route host restored to ${current_host}; verifying HTTP response"
curl -sS --fail --max-time 15 "http://${current_host}" >/dev/null

echo "Route host matches annotation and returns HTTP 200."
