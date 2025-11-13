AGENT_BINARY_NAME = agent
GEVALS_BINARY_NAME = gevals

.PHONY: clean
clean:
	rm -f $(AGENT_BINARY_NAME) $(GEVALS_BINARY_NAME)

.PHONY: build-agent
build-agent: clean
	go build -o $(AGENT_BINARY_NAME) ./cmd/agent

.PHONY: build-gevals
build-gevals: clean
	go build -o $(GEVALS_BINARY_NAME) ./cmd/gevals/

.PHONY: build
build: build-agent build-gevals

.PHONY: test
test:
	go test ./...
