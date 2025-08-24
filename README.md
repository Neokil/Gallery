# Photo Gallery

A simple, password-protected photo sharing website built in Go. Perfect for sharing event photos with guests.

## Features

- **Password Protection**: Single shared password for all users
- **Photo Upload**: Drag-and-drop or click to upload multiple photos
- **Gallery View**: Grid layout with click-to-expand modal
- **Mobile Responsive**: Works great on desktop and mobile
- **Secure**: Session-based auth with security headers
- **Lightweight**: Minimal dependencies, fast performance

## Quick Start

### Local Development

1. **Clone and setup**:
   ```bash
   git clone <your-repo>
   cd photo-gallery
   ```

2. **Set environment variables**:
   ```bash
   export GALLERY_PASSWORD="your-secret-password"
   export SITE_TITLE="My Event Photos"  # Optional
   ```

3. **Run the application**:
   ```bash
   go mod tidy
   go run main.go
   ```

4. **Access the gallery**:
   Open http://localhost:8080 and enter your password.

### Docker Deployment

1. **Using Docker Compose** (recommended):
   ```bash
   # Edit docker-compose.yml to set your password
   docker-compose up -d
   ```

2. **Using Docker directly**:
   ```bash
   docker build -t photo-gallery .
   docker run -p 8080:8080 \
     -e GALLERY_PASSWORD="your-secret-password" \
     -e SITE_TITLE="My Event Photos" \
     -v $(pwd)/uploads:/root/uploads \
     photo-gallery
   ```

## Configuration

Set these environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GALLERY_PASSWORD` | ✅ | - | Password to access the gallery |
| `SITE_TITLE` | ❌ | "Photo Gallery" | Title shown on the website |
| `UPLOAD_DIR` | ❌ | "./uploads" | Directory to store uploaded photos |
| `PORT` | ❌ | "8080" | Server port |

## Security

- Password-protected with secure sessions
- Security headers (XSS protection, content type sniffing, etc.)
- File type validation (only images allowed)
- Should be deployed behind HTTPS in production

## Production Deployment

1. **Use HTTPS**: Deploy behind a reverse proxy (nginx, Caddy, or cloud load balancer)
2. **Set strong password**: Use a long, random password
3. **Backup uploads**: The `uploads` directory contains all photos
4. **Monitor disk space**: Photos are stored on disk

### Example nginx config:
```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## File Structure

```
photo-gallery/
├── main.go              # Main application
├── templates/           # HTML templates
│   ├── login.html
│   └── gallery.html
├── static/              # CSS and JavaScript
│   ├── style.css
│   └── script.js
├── uploads/             # Uploaded photos (created automatically)
├── Dockerfile           # Docker build file
├── docker-compose.yml   # Docker Compose config
└── README.md           # This file
```

## License

MIT License - feel free to use for your events!