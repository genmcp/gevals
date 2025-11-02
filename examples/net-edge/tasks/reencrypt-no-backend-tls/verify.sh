#!/usr/bin/env bash
set -euo pipefail

for bin in oc curl jq; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "${bin} is required for verification" >&2
    exit 1
  fi
done

NAMESPACE="${NAMESPACE:-netedge-scenario4}"
APP_NAME="${APP_NAME:-hello}"

route_json="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o json 2>/dev/null || true)"
if [[ -z "${route_json}" ]]; then
  echo "route ${NAMESPACE}/${APP_NAME} not found" >&2
  exit 1
fi

termination="$(echo "${route_json}" | jq -r '.spec.tls.termination // empty')"
if [[ -n "${termination}" && "${termination}" != "edge" ]]; then
  echo "route ${APP_NAME} still configured for TLS termination (${termination})" >&2
  exit 1
fi

dest_ca="$(echo "${route_json}" | jq -r '.spec.tls.destinationCACertificate // empty')"
if [[ -n "${dest_ca}" ]]; then
  echo "route ${APP_NAME} still carries a destination CA certificate" >&2
  exit 1
fi

host="$(echo "${route_json}" | jq -r '.spec.host // empty')"
if [[ -z "${host}" ]]; then
  echo "route ${APP_NAME} has no admitted host" >&2
  exit 1
fi

echo "Verifying curl succeeds over plain HTTP"
if ! curl -sS --max-time 10 "http://${host}" >/dev/null; then
  echo "curl against http://${host} failed" >&2
  exit 1
fi

echo "Route back to plain HTTP and curl succeeds."
