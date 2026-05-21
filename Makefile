.PHONY: build test lint clean install fmt vet tidy

BINARY_NAME=patchwork
GO=go
GOFLAGS=-buildvcs=false
LDFLAGS=-ldflags "-s -w"

BUILD_DIR=bin
MAIN_CMD=cmd/patchwork/main.go

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(MAIN_CMD)

test:
	$(GO) test $(GOFLAGS) -race -coverprofile=coverage.out -covermode=atomic ./...

test-verbose:
	$(GO) test $(GOFLAGS) -v -race -coverprofile=coverage.out -covermode=atomic ./...

lint:
	@gofmt -l -s . | read && { echo "gofmt found unformatted files"; gofmt -d .; exit 1; } || true
	$(GO) vet $(GOFLAGS) ./...

fmt:
	gofmt -w -s .

vet:
	$(GO) vet $(GOFLAGS) ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

install: build
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME)"

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  test         - Run tests with coverage"
	@echo "  test-verbose - Run tests verbosely with coverage"
	@echo "  lint         - Run gofmt check and go vet"
	@echo "  fmt          - Format code with gofmt"
	@echo "  vet          - Run go vet"
	@echo "  tidy         - Run go mod tidy"
	@echo "  clean        - Remove build artifacts"
	@echo "  install      - Build and install to /usr/local/bin"
	@echo "  uninstall    - Remove from /usr/local/bin"
	@echo "  run          - Build and run"
	@echo "  help         - Show this help"
