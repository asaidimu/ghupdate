# Makefile

APP_NAME := example
OUTPUT_DIR := dist

# Version can be overridden by an environment variable (like from semantic-release)
# or default to git describe if not provided.
VERSION ?= $(shell git describe --tags --abbrev=0 --always)
# Get the current UTC build date in ISO 8601 format
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# The Go module path for your project (e.g., from go.mod)
MODULE_PATH := gh-autoupdate # IMPORTANT: Replace with your actual go.mod module name

# Linker flags to inject version and build date into the binary.
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)

.PHONY: all build test clean

all: build

# Main build target
build:
	@mkdir -p $(OUTPUT_DIR) # Ensure the output directory exists
	@echo "Building $(APP_NAME) for linux/amd64, version $(VERSION)..."
	GOOS=linux GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(APP_NAME)-v$(VERSION)-linux-amd64 \
		./example/main.go
	@echo "Build complete: $(OUTPUT_DIR)/$(APP_NAME)-v$(VERSION)-linux-amd64"

test:
	go test -v ./...

clean:
	rm -rf $(OUTPUT_DIR) # Remove the entire dist directory
