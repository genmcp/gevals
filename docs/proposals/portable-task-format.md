# Proposal: Portable Task Format for MCP Server Evaluation

## Problem Statement

gevals currently uses bash scripts for setup, verification, and cleanup of evaluation tasks. While this approach works, it creates several problems as the project scales:

**Script complexity grows quickly.** What starts as a simple `kubectl get pod` check becomes 90 lines of bash with jq parsing, retry loops, timeout handling, and error messages. These scripts are hard to read, debug, and maintain. See `examples/kube-mcp-server/tasks/create-canary-deployment/verify.sh` or `examples/kube-mcp-server/tasks/setup-dev-cluster/verify.sh` for examples of this complexity.

**Tasks are not portable across evaluation tools.** An MCP server developer who publishes evaluation tasks today is publishing gevals-specific artifacts. A user who wants to run these same task scenarios against their own application using a different evaluation framework (one focused on response quality rather than MCP behavior) cannot easily reuse them. The task definition is tangled with gevals-specific execution.

**Verification lives in two places.** The current design separates "did the task succeed" (verify script in task.yaml) from "did the agent use MCP correctly" (assertions in eval.yaml). This layering exists for good reason (the same task can run against different MCP servers), but it's confusing for developers writing simple evaluations.

**Language barriers limit contribution.** A Go developer who wants to write verification logic must write bash. A Python developer must write bash. Everyone writes bash, regardless of whether it's the right tool for the job. While shebangs technically allow other languages, the lack of structured I/O makes this awkward.

**Registry automation is fragile.** If an MCP registry wants to periodically run evaluations against registered servers and publish quality scores, it needs to execute arbitrary bash scripts. There's no structured contract for what these scripts need or produce.

## Requirements

Any solution must satisfy these constraints:

### Portability

1. Task definitions must be consumable by tools other than gevals. An evaluation framework focused on response quality should be able to load a task's prompt, run it through their own agent, and apply their own quality metrics.

2. Tasks must be self-contained. All artifacts needed to run a task (manifests, expected files, etc.) should be bundled with or referenced by the task definition.

3. Tasks must not require a specific programming language runtime. A task authored by a Go developer should be runnable on a system that only has Python, and vice versa.

### Expressiveness

4. The format must handle the verification patterns present in existing tasks, including:
   - Checking resource state (exists, has condition, field equals value)
   - Executing commands inside containers
   - Making HTTP requests and checking responses
   - Waiting for conditions with timeouts
   - Comparing JSON structures with normalization
   - Verifying stability over time (condition stays true for N seconds)
   - Iterating over multiple items
   - Conditional logic (pass if any of these checks pass)

5. The format must support setup and cleanup with the same expressiveness as verification.

6. There must be an escape hatch for complex cases that don't fit the declarative model.

### Extensibility

7. Domain-specific verification logic (Kubernetes-specific checks, database queries, etc.) must be pluggable without modifying gevals core.

8. Extensions must be distributable and runnable without requiring users to install language-specific toolchains.

9. Extensions must be safe to run in automated environments (registries running evals on untrusted MCP servers).

### Compatibility

10. The solution should preserve the current layered model where task-level verification is separate from MCP-level assertions. This enables running the same tasks against different MCP servers.

11. Existing bash scripts should continue to work during migration.

## Solution Overview

Replace the current script-based task format with a declarative YAML format that:

- Defines setup, verification, and cleanup as sequences of typed steps
- Provides built-in step types for generic operations (shell commands, HTTP requests, file operations)
- Supports extensions for domain-specific logic via a simple binary protocol
- Allows extensions to be imported and used with a short prefix throughout a task
- Keeps scripts as an escape hatch for complex cases

The key insight is that tasks should describe *what* needs to happen, not *how* to make it happen. The "how" is handled by step executors (built-in or extensions) that can be implemented once and reused across many tasks.

### Example

A task that currently looks like this:

```yaml
kind: Task
metadata:
  name: create-pod
steps:
  setup:
    file: setup.sh
  verify:
    file: verify.sh
  cleanup:
    file: cleanup.sh
  prompt:
    inline: "Create a pod named test-pod running nginx"
```

With accompanying bash scripts totaling 50+ lines, becomes:

```yaml
kind: Task
apiVersion: mcp-eval/v1
metadata:
  name: create-pod

spec:
  imports:
    - package: github.com/gevals/ext-kubernetes@v1
      as: k8s

  env:
    NAMESPACE: test-{random.id}
    POD_NAME: test-pod

  prompt: |
    Create a pod named {env.POD_NAME} running nginx in namespace {env.NAMESPACE}

  setup:
    - k8s.create-namespace:
        name: "{env.NAMESPACE}"

  verify:
    - k8s.condition:
        resource: pod/{env.POD_NAME}
        namespace: "{env.NAMESPACE}"
        type: Ready
        status: "True"
        timeout: 60s

  cleanup:
    - k8s.delete-namespace:
        name: "{env.NAMESPACE}"
        wait: false
```

No bash scripts. This is declarative and portable. The kubernetes extension handles the complexity of waiting, retrying, and parsing.

## Detailed Design

### Task Schema

```yaml
kind: Task
apiVersion: mcp-eval/v1

metadata:
  name: string                    # Required. Unique identifier.
  description: string             # Optional. Human-readable description.
  difficulty: easy|medium|hard    # Optional. For categorization.
  timeout: duration               # Optional. Default: 5m. Max time for entire task.
  tags: [string]                  # Optional. For filtering/categorization.

spec:
  imports: [import]               # Optional. Extensions to use.

  requires:                       # Optional. Pre-flight checks.
    - command: string             # Binary that must be in PATH

  env:                            # Optional. Variables available to all steps.
    KEY: value                    # Supports ${{ }} templating

  prompt: string                  # Required. What to tell the agent.

  setup: [step]                   # Optional. Steps to run before agent.
  verify: [step]                  # Required. Steps to check task success.
  cleanup: [step]                 # Optional. Steps to run after (always runs).
```

### Extension Imports

Extensions are imported at the top of a task and given a short alias:

```yaml
spec:
  imports:
    - package: github.com/gevals/ext-kubernetes@v1
      as: k8s
    - package: github.com/gevals/ext-postgres@v2
      as: pg
```

Once imported, extension actions and checks can be used as step types with the alias prefix:

```yaml
setup:
  - k8s.create-namespace:
      name: my-namespace
  - k8s.apply:
      file: artifacts/deployment.yaml

verify:
  - k8s.condition:
      resource: deployment/my-app
      type: Available
  - pg.query:
      sql: "SELECT count(*) FROM users"
      expect:
        value: 5

cleanup:
  - k8s.delete-namespace:
      name: my-namespace
```

This is equivalent to the verbose form:

```yaml
- extension:
    package: github.com/gevals/ext-kubernetes@v1
    action: create-namespace
    args:
      name: my-namespace
```

The short form is preferred for readability. The verbose form remains available for cases where you need additional options like `continueOnError` or `outputs`.

### Variable Templating

Variables use `{name}` syntax and can appear in any string value:

| Expression | Description |
|------------|-------------|
| `{env.NAME}` | Environment variable from `spec.env` |
| `{random.id}` | Random alphanumeric string (8 chars) |
| `{random.port}` | Random available TCP port |
| `{task.name}` | Task metadata |
| `{steps.STEP_ID.outputs.NAME}` | Output from a previous step |
| `{agent.output}` | Agent's response (verify only) |

Environment variables from the shell (`$HOME`, etc.) are also available via `{env.HOMEi}` if not overridden in `spec.env`.

### Built-in Step Types

gevals provides a small set of generic, domain-agnostic step types. Domain-specific operations (Kubernetes, databases, etc.) belong in extensions.

#### command

Runs a shell command.

```yaml
- command:
    id: step-name              # Optional. For referencing outputs.
    run: string                # Required. Command to execute.
    shell: string              # Optional. Default: $SHELL or /bin/sh
    workdir: string            # Optional. Working directory.
    timeout: duration          # Optional. Default: 60s
    continueOnError: boolean   # Optional. Default: false (true for cleanup)
    env:                       # Optional. Additional environment variables.
      KEY: value
    outputs:                   # Optional. Capture values from command.
      varName: "{stdout}"   # Captures stdout
      other: "{stderr}"     # Captures stderr
      code: "{exitCode}"    # Captures exit code
    expect:                    # Optional. For verify phase.
      exitCode: number         # Expected exit code. Default: 0
      stdout:                  # Expected stdout content
        equals: string         # Exact match
        contains: string       # Substring match
        matches: regex         # Regex match
      stderr:                  # Same options as stdout
```

The `expect` block is primarily for verify steps. In setup/cleanup, commands fail if they return non-zero (unless `continueOnError: true`).

#### http

Makes an HTTP request.

```yaml
- http:
    id: step-name
    url: string                # Required. URL to request.
    method: string             # Optional. Default: GET
    headers:                   # Optional.
      Header-Name: value
    body: string               # Optional. Request body.
    timeout: duration          # Optional. Default: 30s
    continueOnError: boolean
    outputs:
      body: "{response.body}"
      status: "{response.status}"
      header: "{response.headers.X-Custom}
    expect:                    # For verify phase.
      status: number           # Expected status code
      body:
        contains: string
        matches: regex
        json:
          path: string         # JSONPath expression
          equals: any          # Expected value
```

#### file

Creates, checks, or removes files.

```yaml
# Create a file (setup)
- file:
    path: /tmp/config.json
    content: |
      {"key": "value"}
    mode: "0644"               # Optional. File permissions.

# Check a file (verify)
- file:
    path: /tmp/output.txt
    expect:
      exists: true
      contains: "success"
      matches: regex
      mode: "0644"

# Remove a file (cleanup)
- file:
    path: /tmp/config.json
    absent: true
```

#### llm

Uses an LLM judge to verify agent output. Only valid in verify phase.

```yaml
- llm:
    contains: "expected information"   # Semantic containment check
    # or
    exact: "expected response"         # Semantic equivalence check
```

This uses the LLM judge configured at the eval level, not the task level. If no judge is configured, this step fails.

#### script

Runs a script file or inline script. This is the escape hatch for complex logic.

```yaml
# File-based
- script:
    file: ./verify-complex.sh

# Inline with shebang
- script:
    inline: |
      #!/usr/bin/env python3
      import json
      import sys
      # Complex verification logic
      print(json.dumps({"passed": True, "reason": "All checks passed"}))
```

When `protocol: json` is specified (see Script Protocol section), the script receives context on stdin and must output JSON.

```yaml
- script:
    file: ./verify.py
    protocol: json
```

### Control Flow Steps

These steps modify the execution flow of verification.

#### foreach

Iterates over a list, running steps for each item.

```yaml
- foreach:
    var: user
    in: ["alice", "bob", "charlie"]
    # or
    in: "{env.USERS}" # If USERS is a JSON array
    steps:
      - k8s.resource-exists:
          resource: serviceaccount/{user}-sa
          namespace: dev-{user}
```

All iterations must pass for the foreach to pass.

#### anyOf

Passes if any of the sub-steps pass.

```yaml
- anyOf:
    - command:
        run: kubectl get hpa -o jsonpath='{.spec.targetCPUUtilizationPercentage}'
        expect:
          stdout: "50"
    - command:
        run: kubectl get hpa -o jsonpath='{.spec.metrics[0].resource.target.averageUtilization}'
        expect:
          stdout: "50"
```

Short-circuits on first success.

#### group

Groups steps with local setup and cleanup. Useful when verification needs temporary fixtures.

```yaml
- group:
    id: connectivity-test
    setup:
      - k8s.apply:
          file: test-fixtures/curl-pod.yaml
      - k8s.wait:
          resource: pod/curl-test
          condition: Ready
          timeout: 60s
    steps:
      - k8s.exec:
          pod: curl-test
          command: ["curl", "-s", "http://target-service"]
          expect:
            exitCode: 0
    cleanup:
      - k8s.delete:
          resource: pod/curl-test
```

The group's cleanup runs regardless of whether its steps pass.

### Extensions

Extensions are standalone executables that implement domain-specific actions and checks. They are distributed as binaries and invoked by gevals via a JSON protocol.

#### Package References

Extensions are referenced by package identifier:

```
github.com/gevals/ext-kubernetes@v1.2.0
github.com/myorg/ext-postgres@v0.1.0
```

gevals downloads the appropriate binary for the current platform from the package's releases and caches it locally (`~/.gevals/extensions/`).

#### Release Structure

Extension authors publish platform-specific binaries:

```
github.com/gevals/ext-kubernetes/releases/v1.2.0/
├── ext-kubernetes-darwin-amd64
├── ext-kubernetes-darwin-arm64
├── ext-kubernetes-linux-amd64
├── ext-kubernetes-linux-arm64
├── checksums.sha256
└── extension.yaml              # Extension manifest
```

#### Extension Manifest

Each extension includes a manifest describing its capabilities:

```yaml
name: kubernetes
version: 1.2.0
description: Kubernetes resource verification

requires:
  - command: kubectl

actions:
  - name: create-namespace
    description: Create a namespace
    args:
      name: { type: string, required: true }
      labels: { type: object, required: false }

  - name: delete-namespace
    description: Delete a namespace
    args:
      name: { type: string, required: true }
      wait: { type: boolean, default: false }

  - name: apply
    description: Apply a manifest file or inline content
    args:
      file: { type: string }
      inline: { type: string }
      namespace: { type: string }

  - name: delete
    description: Delete a resource
    args:
      resource: { type: string, required: true }
      namespace: { type: string, default: default }

  - name: exec
    description: Execute a command in a container
    args:
      pod: { type: string, required: true }
      namespace: { type: string, default: default }
      container: { type: string }
      command: { type: array, required: true }

  - name: wait
    description: Wait for a resource condition
    args:
      resource: { type: string, required: true }
      namespace: { type: string, default: default }
      condition: { type: string, required: true }
      timeout: { type: duration, default: 60s }

checks:
  - name: condition
    description: Check a resource has a condition
    args:
      resource: { type: string, required: true }
      namespace: { type: string, default: default }
      type: { type: string, required: true }
      status: { type: string, default: "True" }
      timeout: { type: duration, default: 60s }

  - name: condition-stable
    description: Check a condition remains true for a duration
    args:
      resource: { type: string, required: true }
      namespace: { type: string, default: default }
      condition: { type: string, required: true }
      duration: { type: duration, required: true }

  - name: resource-exists
    description: Check a resource exists
    args:
      resource: { type: string, required: true }
      namespace: { type: string, default: default }

  - name: resource-matches
    description: Compare a resource against expected state
    args:
      resource: { type: string, required: true }
      namespace: { type: string, default: default }
      expected: { type: string, required: true }
      ignorePaths: { type: array }
```

Actions are used in setup/cleanup phases. Checks are used in verify phase. Some operations (like `exec`) can be both.

#### Invocation Protocol

gevals invokes extensions via command line:

```bash
# For actions (setup/cleanup)
ext-kubernetes action create-namespace --input <json-file>

# For checks (verify)
ext-kubernetes check condition --input <json-file>
```

The input file contains:

```json
{
  "args": {
    "resource": "pod/test-pod",
    "namespace": "default",
    "type": "Ready",
    "status": "True",
    "timeout": "60s"
  },
  "context": {
    "env": {
      "NAMESPACE": "test-abc123"
    },
    "workdir": "/path/to/task"
  }
}
```

Extensions write their result to stdout:

```json
{
  "success": true,
  "message": "Pod test-pod has condition Ready=True",
  "outputs": {
    "actualStatus": "True",
    "lastTransitionTime": "2024-01-15T10:30:00Z"
  }
}
```

For failures:

```json
{
  "success": false,
  "message": "Timed out waiting for condition",
  "error": "Pod test-pod condition Ready=False after 60s"
}
```

Exit codes: 0 for success, non-zero for failure. The JSON output provides details.

#### Sandboxing

For registry automation or other untrusted contexts, extensions can be run in containers:

```yaml
# gevals config
extensions:
  sandbox: docker
  allowedSources:
    - github.com/genmcp/*
    - github.com/my-trusted-org/*
```

When sandboxed, gevals runs the extension binary inside a container with limited capabilities.

### Script Protocol

Scripts can opt into structured I/O by specifying `protocol: json`:

```yaml
- script:
    file: ./verify.py
    protocol: json
```

With this protocol, the script receives context on stdin:

```json
{
  "task": {
    "name": "create-pod",
    "prompt": "Create a pod named test-pod..."
  },
  "agent": {
    "output": "I've created the pod...",
    "exitCode": 0
  },
  "mcp": {
    "callHistory": {
      "toolCalls": [
        {
          "serverName": "kubernetes",
          "toolName": "create_pod",
          "arguments": { "name": "test-pod" },
          "result": { "success": true },
          "timestamp": "2024-01-15T10:30:00Z"
        }
      ],
      "resourceReads": [],
      "promptGets": []
    }
  },
  "env": {
    "NAMESPACE": "test-abc123"
  },
  "steps": {
    "previous-step-id": {
      "outputs": {
        "someValue": "captured output"
      }
    }
  }
}
```

The script must output JSON:

```json
{
  "passed": true,
  "reason": "All checks passed",
  "checks": [
    { "name": "pod-exists", "passed": true, "message": "Pod found" },
    { "name": "pod-running", "passed": true, "message": "Status: Running" }
  ],
  "outputs": {
    "podUid": "abc-123"
  }
}
```

Exit code 0 indicates the script ran successfully (the `passed` field indicates verification result). Non-zero exit indicates script failure (distinct from verification failure).

Scripts without `protocol: json` work as today: exit 0 = pass, non-zero = fail.

### Cleanup Semantics

Cleanup steps have special behavior:

1. **Always run.** Cleanup runs even if setup or verify fails.

2. **All steps run.** If a cleanup step fails, subsequent cleanup steps still run.

3. **continueOnError defaults to true.** Cleanup step failures are logged but don't affect task pass/fail status.

4. **Reverse order.** Cleanup steps run in reverse order of definition (last defined runs first). This matches the common pattern of cleaning up resources in reverse order of creation.

### Eval-Level Configuration

The eval.yaml format gains options for the new task format:

```yaml
kind: Eval
metadata:
  name: kubernetes-eval

config:
  agent:
    type: builtin.claude-code

  mcpConfigFile: ./mcp-config.yaml

  # Extension configuration
  extensions:
    cacheDir: ~/.gevals/extensions    # Default
    timeout: 5m                        # Max time for extension operations
    sandbox: none                      # none | docker
    allowedSources:                    # For sandboxed mode
      - github.com/gevals/*

  # Task configuration
  taskSets:
    - glob: ./tasks/*/*.yaml
      assertions:                      # MCP-level assertions (unchanged)
        toolsUsed:
          - server: kubernetes
        minToolCalls: 1
        maxToolCalls: 20
```

The MCP-level assertions in eval.yaml remain separate from task-level verification. This preserves the ability to run the same tasks against different MCP servers with different assertion requirements.

### Migration Path

Existing tasks using script files continue to work:

```yaml
# This still works
steps:
  setup:
    file: setup.sh
  verify:
    file: verify.sh
  cleanup:
    file: cleanup.sh
```

The new format is opt-in via `apiVersion: mcp-eval/v1` and the `spec` structure. Tasks without `apiVersion` or using the old `steps.setup.file` format use the legacy execution path.

## Appendix A: Built-in Step Reference

### command

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| id | string | No | auto | Step identifier for outputs |
| run | string | Yes | - | Command to execute |
| shell | string | No | $SHELL | Shell to use |
| workdir | string | No | task dir | Working directory |
| timeout | duration | No | 60s | Execution timeout |
| continueOnError | boolean | No | false | Don't fail on error |
| env | map | No | {} | Additional environment |
| outputs | map | No | {} | Output variable capture |
| expect | object | No | exitCode: 0 | Expected results |

### http

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| id | string | No | auto | Step identifier |
| url | string | Yes | - | Request URL |
| method | string | No | GET | HTTP method |
| headers | map | No | {} | Request headers |
| body | string | No | - | Request body |
| timeout | duration | No | 30s | Request timeout |
| continueOnError | boolean | No | false | Don't fail on error |
| outputs | map | No | {} | Output variable capture |
| expect | object | No | status: 2xx | Expected results |

### file

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| path | string | Yes | - | File path |
| content | string | No | - | Content to write (create mode) |
| mode | string | No | 0644 | File permissions |
| absent | boolean | No | false | Ensure file doesn't exist |
| expect | object | No | - | Expected state (verify mode) |

### llm

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| contains | string | Yes* | - | Expected semantic content |
| exact | string | Yes* | - | Expected semantic match |

*One of contains or exact required.

### script

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| file | string | Yes* | - | Script file path |
| inline | string | Yes* | - | Inline script content |
| protocol | string | No | - | "json" for structured I/O |
| timeout | duration | No | 300s | Execution timeout |
| continueOnError | boolean | No | false | Don't fail on error |
| outputs | map | No | {} | Output capture (json protocol) |

*One of file or inline required.

### foreach

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| var | string | Yes | - | Loop variable name |
| in | [any] | Yes | - | Items to iterate |
| steps | [step] | Yes | - | Steps to run for each item |

### anyOf

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| (list of steps) | [step] | Yes | - | Steps where any must pass |

### group

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| id | string | No | auto | Group identifier |
| setup | [step] | No | [] | Setup steps for group |
| steps | [step] | Yes | - | Verification steps |
| cleanup | [step] | No | [] | Cleanup steps for group |

## Appendix B: Kubernetes Extension Reference

The `github.com/gevals/ext-kubernetes` extension provides Kubernetes-specific operations.

### Actions

#### create-namespace

Creates a Kubernetes namespace.

```yaml
- k8s.create-namespace:
    name: my-namespace
    labels:
      env: test
```

#### delete-namespace

Deletes a Kubernetes namespace.

```yaml
- k8s.delete-namespace:
    name: my-namespace
    wait: true              # Wait for deletion to complete
```

#### apply

Applies a manifest from file or inline.

```yaml
- k8s.apply:
    file: artifacts/deployment.yaml
    namespace: default      # Override namespace in manifest

- k8s.apply:
    inline: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: my-config
      data:
        key: value
```

#### delete

Deletes a resource.

```yaml
- k8s.delete:
    resource: deployment/my-app
    namespace: default
```

#### exec

Executes a command in a container.

```yaml
- k8s.exec:
    pod: my-pod
    namespace: default
    container: main         # Optional, defaults to first container
    command: ["cat", "/etc/config"]
    outputs:
      config: "{stdout}"
    expect:
      exitCode: 0
```

#### wait

Waits for a resource condition.

```yaml
- k8s.wait:
    resource: deployment/my-app
    namespace: default
    condition: Available
    timeout: 120s
```

### Checks

#### condition

Verifies a resource has a condition.

```yaml
- k8s.condition:
    resource: pod/my-pod
    namespace: default
    type: Ready
    status: "True"
    timeout: 60s
```

#### condition-stable

Verifies a condition remains true for a duration.

```yaml
- k8s.condition-stable:
    resource: pod/my-pod
    namespace: default
    condition: Ready
    duration: 30s
```

#### resource-exists

Verifies a resource exists.

```yaml
- k8s.resource-exists:
    resource: serviceaccount/my-sa
    namespace: default
```

#### resource-matches

Compares a resource against expected state.

```yaml
- k8s.resource-matches:
    resource: networkpolicy/my-policy
    namespace: default
    expected: artifacts/expected-policy.yaml
    ignorePaths:
      - .metadata.resourceVersion
      - .metadata.uid
      - .metadata.creationTimestamp
```

#### field-equals

Checks a specific field value.

```yaml
- k8s.field-equals:
    resource: deployment/my-app
    namespace: default
    jsonpath: .spec.replicas
    value: 3
```

