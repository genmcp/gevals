AGENT_BINARY_NAME = agent
GEVALS_BINARY_NAME = gevals

# Release build variables (can be overridden)
VERSION ?= dev
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.PHONY: clean
clean:
	rm -f $(AGENT_BINARY_NAME) $(GEVALS_BINARY_NAME)
	rm -f *.zip *.bundle

.PHONY: build-agent
build-agent: clean
	go build -o $(AGENT_BINARY_NAME) ./cmd/agent

.PHONY: build-gevals
build-gevals: clean
	go build -o $(GEVALS_BINARY_NAME) ./cmd/gevals/

.PHONY: build
build: build-agent build-gevals

# Release targets for CI/CD
.PHONY: build-release
build-release:
	@echo "Building release binaries for $(GOOS)/$(GOARCH)..."
	@mkdir -p dist
	@if [ "$(GOOS)" = "windows" ]; then \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags="-s -w" -o "dist/$(GEVALS_BINARY_NAME)-$(GOOS)-$(GOARCH).exe" ./cmd/gevals; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags="-s -w" -o "dist/$(AGENT_BINARY_NAME)-$(GOOS)-$(GOARCH).exe" ./cmd/agent; \
	else \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags="-s -w" -o "dist/$(GEVALS_BINARY_NAME)-$(GOOS)-$(GOARCH)" ./cmd/gevals; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags="-s -w" -o "dist/$(AGENT_BINARY_NAME)-$(GOOS)-$(GOARCH)" ./cmd/agent; \
	fi
	@echo "Build complete!"

.PHONY: package-release
package-release:
	@echo "Packaging release artifacts for $(GOOS)/$(GOARCH)..."
	@cd dist && \
	if [ "$(GOOS)" = "windows" ]; then \
		zip "$(GEVALS_BINARY_NAME)-$(GOOS)-$(GOARCH).zip" "$(GEVALS_BINARY_NAME)-$(GOOS)-$(GOARCH).exe"; \
		zip "$(AGENT_BINARY_NAME)-$(GOOS)-$(GOARCH).zip" "$(AGENT_BINARY_NAME)-$(GOOS)-$(GOARCH).exe"; \
	else \
		zip "$(GEVALS_BINARY_NAME)-$(GOOS)-$(GOARCH).zip" "$(GEVALS_BINARY_NAME)-$(GOOS)-$(GOARCH)"; \
		zip "$(AGENT_BINARY_NAME)-$(GOOS)-$(GOARCH).zip" "$(AGENT_BINARY_NAME)-$(GOOS)-$(GOARCH)"; \
	fi
	@echo "Packaging complete!"

.PHONY: sign-release
sign-release:
	@echo "Signing release artifacts for $(GOOS)/$(GOARCH)..."
	@cd dist && \
	cosign sign-blob --yes "$(GEVALS_BINARY_NAME)-$(GOOS)-$(GOARCH).zip" \
		--bundle "$(GEVALS_BINARY_NAME)-$(GOOS)-$(GOARCH).zip.bundle" && \
	cosign sign-blob --yes "$(AGENT_BINARY_NAME)-$(GOOS)-$(GOARCH).zip" \
		--bundle "$(AGENT_BINARY_NAME)-$(GOOS)-$(GOARCH).zip.bundle"
	@echo "Signing complete!"

.PHONY: release
release: build-release package-release sign-release
	@echo "Release build complete for $(GOOS)/$(GOARCH)!"

