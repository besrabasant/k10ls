# Binary Name
BINARY_NAME = k10ls

# Go Directories
SRC_DIR = .
BUILD_DIR = ./bin
MAIN_FILE = main.go

# Go commands
GO = go
GOBUILD = $(GO) build
GOTEST = $(GO) test
GOLINT = golangci-lint
GOTIDY = $(GO) mod tidy
GODOWNLOAD = $(GO) mod download
GOFMT = $(GO) fmt

# Build the binary
build: tidy fmt lint
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the tool
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

# Format Go files
fmt:
	@echo "Formatting code..."
	@$(GOFMT) $(SRC_DIR)

# Run Linter (ensure you have golangci-lint installed)
lint:
	@echo "Linting code..."
	@$(GOLINT) run ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@$(GO) get ./...

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."

# Tidy up Go modules
tidy:
	@echo "Tidying modules..."
	@$(GOTIDY)

# Run tests
test:
	@echo "Running tests..."
	@$(GOTEST) ./...

# Help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  build     - Build the application"
	@echo "  run       - Run the application"
	@echo "  fmt       - Format Go source code"
	@echo "  lint      - Run linter (requires golangci-lint)"
	@echo "  deps      - Install dependencies"
	@echo "  clean     - Remove built artifacts"
	@echo "  tidy      - Run go mod tidy"
	@echo "  test      - Run tests"
	@echo "  help      - Show this help message"
