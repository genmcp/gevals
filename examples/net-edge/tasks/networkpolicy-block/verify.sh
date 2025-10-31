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

NAMESPACE="${NAMESPACE:-netedge-scenario3}"
APP_NAME="${APP_NAME:-hello}"
NP_NAME="${NP_NAME:-deny-router}"

route_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.host}' 2>/dev/null || true)"
if [[ -z "${route_host}" ]]; then
  echo "route ${NAMESPACE}/${APP_NAME} has no admitted host" >&2
  exit 1
fi

echo "Route host restored to ${route_host}; verifying HTTP response"
if ! curl -sS --fail --max-time 15 "http://${route_host}" >/dev/null; then
  echo "route ${route_host} still failing" >&2
  oc -n "${NAMESPACE}" get networkpolicy "${NP_NAME}" >/dev/null 2>&1 && echo "networkpolicy ${NAMESPACE}/${NP_NAME} still present" >&2
  exit 1
fi

np_ingress="$(oc -n "${NAMESPACE}" get networkpolicy "${NP_NAME}" -o jsonpath='{.spec.ingress}' 2>/dev/null || true)"
if [[ -n "${np_ingress}" ]]; then
  if [[ "${np_ingress}" == "[]" ]]; then
    echo "networkpolicy ${NAMESPACE}/${NP_NAME} still configured as default-deny" >&2
    exit 1
  else
    echo "NetworkPolicy ${NP_NAME} still exists but ingress rules are configured."
  fi
fi

if [[ -z "${np_ingress}" ]]; then
  echo "NetworkPolicy removed and Route responds successfully."
else
  echo "Route responds successfully with adjusted NetworkPolicy ingress rules."
fi
