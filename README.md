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
- **Automatic metadata generation**: Creates metadata for existing images on startup
- **EXIF photo time extraction**: Extracts actual photo taken time from image metadata
  - Supports JPEG images with EXIF DateTime tags (DateTimeOriginal, DateTime, DateTimeDigitized)
  - Filename pattern recognition for images without EXIF data
  - Supports common camera/phone filename patterns (IMG_20231225_143022, PXL_20231225_143022, etc.)
  - Gracefully falls back to upload time when no date information is available
- **Smart photo sorting**: Orders photos by actual photo time (newest first), falls back to upload time
- **Automatic thumbnail generation**: Creates 300px thumbnails for fast gallery loading
  - Thumbnails generated on upload and startup for existing images
  - Maintains aspect ratio with high-quality JPEG compression
  - Falls back to original image if thumbnail unavailable
  - Automatic cleanup of orphaned thumbnails on startup
- **Metadata cleanup**: Removes orphaned metadata files automatically
- **Responsive design**: Works on desktop and mobile
- **Dark mode support**: Automatic based on system preference

## API Endpoints

- `GET /` - Gallery page with photo grid and filters
- `GET /login` - Login page
- `POST /login` - Authentication
- `POST /upload` - Upload photos with metadata
- `GET /download-all` - Download photos as ZIP (supports filtering)
- `GET /uploads/{filename}` - Serve uploaded photos (full resolution)
- `GET /thumbnails/{filename}` - Serve photo thumbnails (300px max)
- `GET /static/{filename}` - Serve static assets

## Environment Variables

- `GALLERY_PASSWORD` - Required. Password for accessing the gallery
- `SITE_TITLE` - Optional. Title displayed on pages (default: "Photo Gallery")
- `UPLOAD_DIR` - Optional. Directory for uploaded photos (default: "./uploads")
- `METADATA_DIR` - Optional. Directory for photo metadata (default: "./metadata")
- `PORT` - Optional. Server port (default: "8080")

## Development

### Prerequisites

- Go 1.24+
- oapi-codegen v2
- Docker and Docker Compose

### Running with Docker Compose

1. **Generate SSL certificates** (for HTTPS support):
   ```bash
   ./generate-ssl.sh
   ```

2. **Start the application**:
   ```bash
   docker-compose up --build -d
   ```

3. **Access the gallery**:
   - HTTP: http://localhost (redirects to HTTPS)
   - HTTPS: https://localhost

### HTTPS Setup

The application includes a reverse proxy setup with Nginx for HTTPS support:

- **Self-signed certificates**: Use `./generate-ssl.sh` for development
- **Production certificates**: Replace files in `nginx/ssl/` with real certificates
- **Let's Encrypt**: You can integrate certbot for automatic certificate management

#### Production SSL Setup

For production, replace the self-signed certificates:

1. Obtain SSL certificates from your certificate authority
2. Place them in the `nginx/ssl/` directory:
   - `cert.pem` - Your SSL certificate
   - `key.pem` - Your private key
3. Restart the containers: `docker-compose restart nginx`

#### Let's Encrypt Integration

For automatic SSL certificates with Let's Encrypt, you can extend the docker-compose with certbot:

```yaml
certbot:
  image: certbot/certbot
  volumes:
    - ./nginx/ssl:/etc/letsencrypt
  command: certonly --webroot --webroot-path=/var/www/certbot --email your-email@domain.com --agree-tos --no-eff-email -d your-domain.com
```

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
