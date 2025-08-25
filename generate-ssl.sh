#!/bin/bash

# Create SSL directory
mkdir -p nginx/ssl

# Generate DH parameters (this may take a while)
echo "Generating DH parameters..."
openssl dhparam -out nginx/dhparam.pem 2048

# Generate self-signed certificate
echo "Generating self-signed SSL certificate..."
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout nginx/ssl/key.pem \
    -out nginx/ssl/cert.pem \
    -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

# Set proper permissions
chmod 600 nginx/ssl/key.pem
chmod 644 nginx/ssl/cert.pem
chmod 644 nginx/dhparam.pem

echo "SSL certificates generated successfully!"
echo "You can now run: docker-compose up --build -d"
echo ""
echo "Access your gallery at:"
echo "  HTTP:  http://192.168.1.241"
echo "  HTTPS: https://192.168.1.241"
echo ""
echo "Note: You'll get a browser warning about the self-signed certificate."
echo "For production, replace the certificates in nginx/ssl/ with real ones."