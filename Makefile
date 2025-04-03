.PHONY: build run clean test lint

# Build information
BINARY_NAME=collector
BUILD_DIR=build
CMD_DIR=cmd/collector

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOLINT=golangci-lint

# Determine the OS
GOOS=$(shell go env GOOS)

all: deps build

deps:
	$(GOMOD) download
	$(GOMOD) tidy

build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

run-config: build
	$(BUILD_DIR)/$(BINARY_NAME) --config config/default.json

run-stdout: build
	$(BUILD_DIR)/$(BINARY_NAME) --stdout --color

clean:
	rm -rf $(BUILD_DIR)

test:
	$(GOTEST) -v ./...

lint:
	$(GOVET) ./...
	$(GOLINT) run ./...

coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Build for various platforms
build-all: build-linux build-windows build-macos

build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)

build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)

build-macos:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-macos-amd64 ./$(CMD_DIR)