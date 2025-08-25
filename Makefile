# Generate server code from OpenAPI spec
generate:
	@echo "Installing oapi-codegen..."
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "Generating server code..."
	mkdir -p internal/api
	oapi-codegen -config api/server-config.yaml api/openapi.yaml

# Build the application
build: generate
	go build -o server ./cmd/server
