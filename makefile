# Project variables
BINARY_NAME := rosy-nigh
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get

# Directories
SRC_DIR := ./cmd/main/main.go
BUILD_DIR := ./build/bin

# Build flags
LDFLAGS := -ldflags="-s -w"

# Debug flags
LDDEBUG := -gcflags="all=-N -l"

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)

# Build the application with debug flags
debug:
	@echo "Building $(BINARY_NAME) with debug flags..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDDEBUG) -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)

# Run the tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v ./...

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@$(GOGET) -v ./...

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME) --verbose --session --debug fuzz -i ./testdata/0x0addedfee0e8a65c9a60067b9fe0f24af96da51d_reentrancy/0.6.12 
.PHONY: all build test clean deps run
