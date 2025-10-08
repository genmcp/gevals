CLI_BINARY_NAME = gevals

.PHONY: clean
clean:
	rm -f $(CLI_BINARY_NAME)

.PHONY: build
build: clean
	go build -o $(CLI_BINARY_NAME) ./cmd/gevals

