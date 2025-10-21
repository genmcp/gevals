AGENT_BINARY_NAME = agent
RUNNER_BINARY_NAME = runner

.PHONY: clean
clean:
	rm -f $(AGENT_BINARY_NAME) $(RUNNER_BINARY_NAME)

.PHONY: build-agent
build-agent: clean
	go build -o $(AGENT_BINARY_NAME) ./cmd/agent

.PHONY: build-runner
build-runner: clean
	go build -o $(RUNNER_BINARY_NAME) ./cmd/runner

.PHONY: build
build: build-agent build-runner

