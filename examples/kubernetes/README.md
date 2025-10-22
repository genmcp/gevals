# Kubernetes Eval Example

This example demonstrates how to configure and run evaluations for Claude Code with Kubernetes MCP server integration.

## Overview

This eval tests Claude Code's ability to create a Kubernetes pod using the Kubernetes MCP server. The eval:
1. Sets up a test namespace
2. Prompts Claude Code to create an nginx pod
3. Verifies the pod is running
4. Cleans up the resources

## Prerequisites

- Kubernetes cluster (e.g., kind, minikube, or any cluster)
- kubectl configured to access your cluster
- Kubernetes MCP server running at `http://localhost:8080/mcp`
- Claude Code installed and available in your PATH

## Directory Structure

```
kubernetes/
├── README.md           # This file
├── eval.yaml          # Main eval configuration
├── agent.yaml         # Claude Code agent configuration
├── mcp-config.yaml    # MCP server configuration
└── tasks/
    ├── create-pod.yaml    # Task definition
    ├── setup.sh          # Setup script (creates namespace)
    ├── verify.sh         # Verification script (checks pod is ready)
    └── cleanup.sh        # Cleanup script (deletes pod)
```

## Configuration Files

### eval.yaml

The main evaluation configuration that references:
- `agentFile`: Path to the Claude Code agent configuration
- `mcpConfigFile`: Path to the MCP server configuration
- `taskSets`: Array of task sets to run with optional assertions

This example includes assertions that:
- Verifies Claude Code uses Kubernetes pod-related tools
- Ensures between 1-10 tool calls are made

### agent.yaml

Defines how to invoke Claude Code with:
- MCP config argument template
- Allowed tools argument template
- The full command to run Claude Code non-interactively

### mcp-config.yaml

Specifies the Kubernetes MCP server:
- Server name: `kubernetes`
- URL: `http://localhost:8080/mcp`
- All tools enabled for this eval

### tasks/create-pod.yaml

Defines the task with:
- **Setup**: Creates a clean namespace for the test
- **Prompt**: Instructs Claude Code to create an nginx pod
- **Verify**: Waits for the pod to be ready (up to 120s)
- **Cleanup**: Deletes the pod after the test

## Running the Eval

From this directory, run:

```bash
# Assuming you've built the gevals binary
../../gevals eval.yaml
```

Or from the repository root:

```bash
./gevals examples/kubernetes/eval.yaml
```

## Expected Behavior

1. The setup script creates the `create-pod-test` namespace
2. Claude Code receives the prompt and uses the Kubernetes MCP server to create the pod
3. The verify script waits for the pod to reach Ready state
4. Assertions check that appropriate Kubernetes tools were used
5. The cleanup script removes the pod

## Customizing

You can extend this example by:

1. **Adding more tasks**: Create additional YAML files in `tasks/` and reference them in `eval.yaml`
2. **Modifying assertions**: Update the `assertions` section in `eval.yaml` to test for specific tool usage patterns
3. **Using inline scripts**: Instead of separate `.sh` files, you can use `inline:` in the task YAML
4. **Testing different scenarios**: Create tasks for deployments, services, configmaps, etc.

## Example: Using Inline Scripts

Instead of separate files, you can define scripts inline in the task YAML:

```yaml
kind: Task
metadata:
  name: "create-nginx-pod"
  difficulty: easy
steps:
  setup:
    inline: |-
      #!/usr/bin/env bash
      kubectl delete namespace create-pod-test --ignore-not-found
      kubectl create namespace create-pod-test
  verify:
    inline: |-
      #!/usr/bin/env bash
      kubectl wait --for=condition=Ready pod/web-server -n create-pod-test --timeout=120s
  cleanup:
    inline: |-
      #!/usr/bin/env bash
      kubectl delete pod web-server -n create-pod-test --ignore-not-found
  prompt:
    inline: Please create a nginx pod named web-server in the create-pod-test namespace
```

## Troubleshooting

- **MCP server not found**: Ensure the Kubernetes MCP server is running at `http://localhost:8080/mcp`
- **kubectl errors**: Verify your kubeconfig is set up correctly
- **Claude Code not found**: Ensure Claude Code is installed and in your PATH
- **Permission errors**: Make sure the shell scripts are executable (`chmod +x tasks/*.sh`)
