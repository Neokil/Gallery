.PHONY: generate clean build run

# Generate server code from OpenAPI spec
generate:
	@echo "Installing oapi-codegen..."
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "Generating server code..."
	mkdir -p internal/api
	oapi-codegen -config api/server-config.yaml api/openapi.yaml

# Clean generated files
clean:
	rm -rf internal/api/generated.go

# Build the application
build: generate
	go build -o bin/photo-gallery ./cmd/server

# Run the application
run: generate
	go run ./cmd/server

# Install dependencies
deps:
	go mod tidy
	go mod download

# Format code
fmt:
	go fmt ./...

# Run tests
test:
	go test ./...