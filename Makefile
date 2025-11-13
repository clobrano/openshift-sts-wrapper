.PHONY: build test clean install fmt vet

BINARY_NAME=openshift-sts-wrapper
INSTALL_PATH=/usr/local/bin

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) .

test:
	@echo "Running tests..."
	@go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@rm -rf artifacts/ _output/ manifests/ tls/

install: build
	@echo "Installing to $(INSTALL_PATH)..."
	@cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Vetting code..."
	@go vet ./...

check: fmt vet test

all: check build
