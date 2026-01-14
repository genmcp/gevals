.PHONY: build clean

BUILD_DIR := bin

build: $(BUILD_DIR)/kubernetes-extension

$(BUILD_DIR)/kubernetes-extension: clean kubernetes-extension/cmd/main.go kubernetes-extension/pkg/extension/extension.go
	@mkdir -p $(BUILD_DIR)
	cd kubernetes-extension && go build -o ../$(BUILD_DIR)/kubernetes-extension ./cmd

clean:
	rm -rf $(BUILD_DIR)
