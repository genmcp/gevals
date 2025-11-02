#!/usr/bin/env bash
set -euo pipefail

for bin in oc jq; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "${bin} is required for verification" >&2
    exit 1
  fi
done

GATEWAY_NAMESPACE="${GATEWAY_NAMESPACE:-netedge-scenario6}"
BACKEND_NAMESPACE="${BACKEND_NAMESPACE:-netedge-scenario6-backend}"
APP_NAME="${APP_NAME:-hello}"

if ! oc -n "${BACKEND_NAMESPACE}" get referencegrant allow-"${APP_NAME}" >/dev/null 2>&1; then
  echo "ReferenceGrant allow-${APP_NAME} missing in ${BACKEND_NAMESPACE}" >&2
  exit 1
fi

route_json="$(oc -n "${GATEWAY_NAMESPACE}" get httproute "${APP_NAME}" -o json 2>/dev/null || true)"
if [[ -z "${route_json}" ]]; then
  echo "HTTPRoute ${GATEWAY_NAMESPACE}/${APP_NAME} not found" >&2
  exit 1
fi

resolved="$(printf '%s' "${route_json}" | jq -r '.status.parents[]?.conditions[]? | select(.type=="ResolvedRefs") | .status' 2>/dev/null || true)"
if [[ "${resolved}" != *"True"* ]]; then
  echo "HTTPRoute ${APP_NAME} still reports ResolvedRefs != True (value: ${resolved:-<none>})" >&2
  exit 1
fi

accepted="$(printf '%s' "${route_json}" | jq -r '.status.parents[]?.conditions[]? | select(.type=="Accepted") | .status' 2>/dev/null || true)"
if [[ -n "${accepted}" && "${accepted}" != *"True"* ]]; then
  echo "HTTPRoute ${APP_NAME} still not accepted by Gateway (Accepted=${accepted})" >&2
  exit 1
fi

echo "ReferenceGrant present and HTTPRoute reports ResolvedRefs=True (and Accepted if available)."
