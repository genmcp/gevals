#!/usr/bin/env bash
set -euo pipefail

for bin in oc curl jq; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "${bin} is required for verification" >&2
    exit 1
  fi
done

NAMESPACE="${NAMESPACE:-netedge-scenario5}"
APP_NAME="${APP_NAME:-hello}"
LOAD_BALANCER_CLASS="${LOAD_BALANCER_CLASS:-example.com/unsupported}"

svc_json="$(oc -n "${NAMESPACE}" get svc "${APP_NAME}" -o json 2>/dev/null || true)"
if [[ -z "${svc_json}" ]]; then
  echo "service ${NAMESPACE}/${APP_NAME} not found" >&2
  exit 1
fi

svc_type="$(printf '%s' "${svc_json}" | jq -r '.spec.type')"
svc_class="$(printf '%s' "${svc_json}" | jq -r '.spec.loadBalancerClass // empty')"

if [[ "${svc_type}" == "LoadBalancer" ]]; then
  if [[ "${svc_class}" == "${LOAD_BALANCER_CLASS}" ]]; then
    echo "service ${NAMESPACE}/${APP_NAME} still using unsupported loadBalancerClass ${LOAD_BALANCER_CLASS}" >&2
    exit 1
  fi

  ingress_host="$(printf '%s' "${svc_json}" | jq -r '.status.loadBalancer.ingress[0].hostname // empty')"
  ingress_ip="$(printf '%s' "${svc_json}" | jq -r '.status.loadBalancer.ingress[0].ip // empty')"
  if [[ -z "${ingress_host}" && -z "${ingress_ip}" ]]; then
    echo "service ${NAMESPACE}/${APP_NAME} LoadBalancer still lacks an external endpoint" >&2
    exit 1
  fi
else
  if [[ "${svc_class}" == "${LOAD_BALANCER_CLASS}" ]]; then
    echo "service ${NAMESPACE}/${APP_NAME} still using unsupported loadBalancerClass ${LOAD_BALANCER_CLASS}" >&2
    exit 1
  fi
fi

route_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.host}' 2>/dev/null || true)"
if [[ -z "${route_host}" ]]; then
  echo "route ${NAMESPACE}/${APP_NAME} has no admitted host" >&2
  exit 1
fi

echo "Route host restored to ${route_host}; verifying HTTP response"
curl -sS --fail --max-time 15 "http://${route_host}" >/dev/null

echo "Service no longer uses unsupported LoadBalancer settings and Route responds successfully."
