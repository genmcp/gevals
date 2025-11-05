# NetEdge Scenario 1 (Service Selector Mismatch)

Evaluate the NetEdge gen-mcp server **Route → Service selector mismatch** scenario with the `gevals`
framework and the Codex GPT-5 coding agent.

## Layout

```text
net-edge/
├── README.md                         # This file
├── mcp-config.yaml                   # Launches the gen-mcp NetEdge server for gevals
├── codex-agent/
│   ├── agent.yaml                    # Codex CLI wiring
│   └── eval.yaml                     # Eval definition (scenario 1)
└── tasks/
    └── selector-mismatch/            # Task definition and helper scripts
        ├── selector-mismatch.yaml
        ├── setup.sh
        ├── verify.sh
        └── cleanup.sh
```

## Prerequisites

- A sibling checkout of [`gen-mcp`](../gen-mcp) with the `genmcp` binary built (`make build`).
- `oc` and `jq` available in `PATH`.
- Access to an OpenShift cluster where the NetEdge tools can deploy the `netedge-scenario1` namespace.
- Codex CLI with an API key that can use the `gpt-5-codex` profile (the eval boots a temporary Codex config with the `rmcp_client` feature flag enabled for HTTP MCP).

Codex requires API-key-based auth for the GPT-5 profile. Start any custom config with:

```toml
preferred_auth_method = "apikey"
```

Example snippet for `~/.codex/config.toml` (update paths as needed):

```toml
preferred_auth_method = "apikey"

[profiles.gpt-5-codex]
model = "gpt-5-codex"

[mcp_servers.netedge]
command = "/Users/you/workspace/gen-mcp/genmcp"
args    = ["run", "-f", "/Users/you/workspace/gen-mcp/examples/netedge-tools/mcpfile.yaml"]
```

The eval overrides the MCP server connection at runtime, pointing Codex at the HTTP proxy that `gevals`
launches for the stdio NetEdge server. Modern Codex builds accept direct HTTP MCP connections (see
the [Codex MCP guide](https://developers.openai.com/codex/mcp)), so no external shim is required,
but Codex still needs a profile that can use the API key.

Provide the key at runtime, for example:

```bash
export OPENAI_API_KEY=sk-...
```

## Running the eval

1. Build the project (from repo root): `make build`
2. Ensure your current shell can reach the OpenShift cluster (`oc whoami` should succeed).
3. Ensure `OPENAI_API_KEY` is exported in the shell that will launch `gevals`.
4. Run the evaluation:

 ```bash
 ./gevals run examples/net-edge/codex-agent/eval.yaml
 ```

`setup.sh` deploys the hello workload, then intentionally breaks the Service selector so the Route loses its
endpoints. The Codex agent must diagnose and repair the mismatch, after which `verify.sh` confirms the selector
and endpoints are healthy. Results are written to `gevals-netedge-selector-mismatch-out.json` by default.

For advanced debugging tips refer to `docs/dev/DEV_DEBUGGING_NOTES.md` in the repo root.
