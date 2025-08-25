# Photo Gallery

A simple photo gallery application built with Go, using OpenAPI specification and oapi-codegen for API generation.

## Architecture

The application has been refactored to use a clean architecture with the following structure:

```
├── api/
│   ├── openapi.yaml          # OpenAPI 3.0 specification
│   └── server-config.yaml    # oapi-codegen configuration
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── api/
│   │   └── generated.go      # Generated API code from OpenAPI spec
│   ├── handlers/
│   │   └── handlers.go       # HTTP handlers implementing the API
│   ├── middleware/
│   │   └── auth.go           # Authentication middleware
│   └── service/
│       ├── auth.go           # Authentication service
│       └── gallery.go        # Gallery business logic
├── static/                   # Static assets (CSS, JS, images)
├── templates/                # HTML templates
└── uploads/                  # Uploaded photos (created at runtime)
```

## Features

- **OpenAPI-driven development**: API specification defines the contract
- **Generated server code**: Uses oapi-codegen with Chi router and strict settings
- **Session-based authentication**: Secure login with password protection
- **Photo upload**: Multi-file upload with metadata (uploader name, event)
- **Photo filtering**: Filter by event or uploader
- **Bulk download**: Download all or filtered photos as ZIP
- **Responsive design**: Works on desktop and mobile
- **Dark mode support**: Automatic based on system preference

## API Endpoints

- `GET /` - Gallery page with photo grid and filters
- `GET /login` - Login page
- `POST /login` - Authentication
- `GET /logout` - Logout and clear session
- `POST /upload` - Upload photos with metadata
- `GET /download-all` - Download photos as ZIP (supports filtering)
- `GET /uploads/{filename}` - Serve uploaded photos
- `GET /static/{filename}` - Serve static assets

## Environment Variables

- `GALLERY_PASSWORD` - Required. Password for accessing the gallery
- `SITE_TITLE` - Optional. Title displayed on pages (default: "Photo Gallery")
- `UPLOAD_DIR` - Optional. Directory for uploaded photos (default: "./uploads")
- `METADATA_DIR` - Optional. Directory for photo metadata (default: "./metadata")
- `PORT` - Optional. Server port (default: "8080")

## Development

### Prerequisites

- Go 1.21+
- oapi-codegen v2

### Setup

1. Install oapi-codegen:
```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

2. Generate API code:
```bash
make generate
```

3. Build and run:
```bash
make build
make run
```

Or run directly:
```bash
GALLERY_PASSWORD=mypassword go run ./cmd/server
```

### Available Make Commands

- `make generate` - Generate server code from OpenAPI spec
- `make build` - Build the application
- `make run` - Generate code and run the application
- `make clean` - Clean generated files
- `make deps` - Install dependencies
- `make fmt` - Format code
- `make test` - Run tests

## Technology Stack

- **Backend**: Go with Chi router
- **API Generation**: oapi-codegen with strict server settings
- **Authentication**: Gorilla Sessions
- **Frontend**: Vanilla HTML/CSS/JavaScript
- **File Upload**: Multipart form handling
- **Archive**: ZIP file generation for bulk downloads

## Security Features

- Session-based authentication
- CSRF protection via session validation
- Secure headers (X-Frame-Options, X-Content-Type-Options, etc.)
- File type validation for uploads
- Path traversal protection

## OpenAPI Integration

The application uses OpenAPI 3.0 specification to:
- Define API contracts
- Generate type-safe server code
- Ensure consistent request/response handling
- Enable API documentation and tooling

The generated code provides:
- Request/response type definitions
- Parameter validation
- Route handling with Chi router
- Strict server interface for type safety