# Progress Overview

## Objective
- Reproduce the gen-mcp NetEdge scenario 1 inside the `gevals` framework using the Codex GPT-5 agent.
- Inline the scenario setup/cleanup scripts so the eval no longer depends on `netedge-break-repair.sh`.
- Wire Codex authentication and MCP connectivity in an automated eval flow.

## Work Completed
- Added self-contained setup/verify/cleanup scripts for the selector-mismatch scenario.
- Added a parallel task bundle for the NXDOMAIN Route host scenario, including staged breakage (`nxdomain-host`) and an eval config that asserts DNS probing via MCP.
- Added the NetworkPolicy-block scenario (scenario 3) with task scripts and an eval config requiring `inspect_route`.
- Added the LoadBalancer-missing scenario (scenario 5) with task scripts and eval config.
- Added the ReferenceGrant-missing scenario (scenario 6) with Gateway/HTTPRoute setup and eval config.
- Added the reencrypt-without-backend-TLS scenario (scenario 4) with task scripts and eval config.
- Created a Codex agent spec that generates a temporary Codex config on each run (trust entries, rmcp_client feature, reasoning high, etc.).
- Introduced `GEVALS_DEBUG=1` support: failing runs now preserve the temp Codex home, config, prompt, and a JSON event log for the Codex CLI.
- Confirmed gevals’ MCP proxy launches the stdio NetEdge server and exposes it over local HTTP (`http://localhost:<port>/mcp`) for the agent.

## Current Status
- Codex CLI now authenticates non-interactively via generated `auth.json` and runs with `danger-full-access`, so `oc` commands succeed during eval runs.
- Kubeconfig propagation ensures the temporary Codex home targets the same cluster as the harness.
- The selector-mismatch scenario passes end-to-end: the agent uses `inspect_route`, patches the Service selector, verifies endpoints, and curls the Route successfully.
- The NXDOMAIN scenario is staged and ready for evaluation with tool assertions targeting `probe_dns_local`.
- The NetworkPolicy block scenario is staged; agent eval pending to validate remediation workflow.
- `geval view` renders concise summaries, assertion status, and tool-call outputs (including the NetEdge response bodies).

## Diagnostics Available
- Run with `GEVALS_DEBUG=1 OPENAI_API_KEY=… ./gevals run …` to preserve Codex config, prompt, logs, and the captured kubeconfig copy.
- `geval view <results.json>` now surfaces timelines and tool outputs; use flags such as `--max-events` to tune verbosity.

## Next Steps
- Harden remaining NetEdge scenarios (Gateway listener conflicts, TLS policy trust failures, etc.) using the same harness.
- Expand assertions as we gain confidence (e.g., require specific tool sequences per scenario).
- Polish docs/instructions so others can reproduce the automated eval flow.
