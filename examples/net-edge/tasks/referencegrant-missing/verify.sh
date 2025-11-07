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

grant_name="$(
  BACKEND_NS="${BACKEND_NAMESPACE}" FROM_NS="${GATEWAY_NAMESPACE}" SVC_NAME="${APP_NAME}" \
  python - <<'PY'
import json
import os
import subprocess
import sys

backend_ns = os.environ.get("BACKEND_NS", "")
from_ns = os.environ.get("FROM_NS", "")
svc_name = os.environ.get("SVC_NAME", "")

cmd = ["oc", "-n", backend_ns, "get", "referencegrant", "-o", "json"]
try:
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
except subprocess.TimeoutExpired:
    print("Timeout while fetching ReferenceGrant resources", file=sys.stderr)
    sys.exit(67)
if result.returncode != 0:
    message = result.stderr.strip() or result.stdout.strip() or "Failed to fetch ReferenceGrant resources"
    print(message, file=sys.stderr)
    sys.exit(65)

raw = result.stdout.strip()
if not raw:
    sys.exit(0)

try:
    payload = json.loads(raw)
except json.JSONDecodeError as exc:
    print(f"JSON decode error: {exc}", file=sys.stderr)
    sys.exit(66)

for item in payload.get("items", []):
    spec = item.get("spec") or {}
    from_entries = spec.get("from") or []
    to_entries = spec.get("to") or []

    allows_from = any(
        (entry or {}).get("group", "") == "gateway.networking.k8s.io"
        and (entry or {}).get("kind", "") == "HTTPRoute"
        and (entry or {}).get("namespace", "") == from_ns
        for entry in from_entries
    )

    allows_to = any(
        ((entry or {}).get("group") or "") in ("", "core")
        and (entry or {}).get("kind", "") == "Service"
        and (entry or {}).get("name", "") == svc_name
        for entry in to_entries
    )

    if allows_from and allows_to:
        print((item.get("metadata") or {}).get("name", ""))
        sys.exit(0)
PY
)"

status=$?
if [[ ${status} -eq 65 ]]; then
  echo "Unable to fetch ReferenceGrant resources in ${BACKEND_NAMESPACE}" >&2
  exit 1
elif [[ ${status} -eq 66 ]]; then
  echo "Failed to parse ReferenceGrant JSON in ${BACKEND_NAMESPACE}" >&2
  exit 1
elif [[ ${status} -eq 67 ]]; then
  echo "Timeout while fetching ReferenceGrant resources in ${BACKEND_NAMESPACE}" >&2
  exit 1
fi

if [[ -z "${grant_name}" ]]; then
  echo "No ReferenceGrant in ${BACKEND_NAMESPACE} permits HTTPRoute from ${GATEWAY_NAMESPACE} to Service ${APP_NAME}" >&2
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
