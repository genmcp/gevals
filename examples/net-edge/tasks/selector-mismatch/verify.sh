#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-netedge-scenario1}"
APP_NAME="${APP_NAME:-hello}"
EXPECTED_LABEL="${APP_LABEL:-hello}"

if ! command -v oc >/dev/null 2>&1; then
  echo "oc CLI is required for verification" >&2
  exit 1
fi

selector_app="$(oc -n "${NAMESPACE}" get svc "${APP_NAME}" -o jsonpath='{.spec.selector.app}' 2>/dev/null || true)"
if [[ "${selector_app}" != "${EXPECTED_LABEL}" ]]; then
  echo "service selector is '${selector_app}', expected '${EXPECTED_LABEL}'" >&2
  exit 1
fi

endpoint_ip="$(oc -n "${NAMESPACE}" get endpoints "${APP_NAME}" -o jsonpath='{.subsets[0].addresses[0].ip}' 2>/dev/null || true)"
if [[ -z "${endpoint_ip}" ]]; then
  echo "endpoints for ${NAMESPACE}/${APP_NAME} are still empty" >&2
  exit 1
fi

echo "Service selector restored and endpoints populated (first IP ${endpoint_ip})."
