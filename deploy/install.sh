#!/bin/bash
# VinzHub API v2 - Deployment Script
# Run this on VPS after cloning the repo

set -e

echo "=== VinzHub API v2 Deployment ==="

# Create directory
mkdir -p /opt/vinzhub/data
mkdir -p /opt/vinzhub/static

# Copy files
cp vinzhub-api /opt/vinzhub/
cp .env.example /opt/vinzhub/.env
cp -r static/* /opt/vinzhub/static/ 2>/dev/null || true

# Set permissions
chmod +x /opt/vinzhub/vinzhub-api

# Install systemd service
cp deploy/vinzhub-api.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable vinzhub-api

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Next steps:"
echo "1. Edit /opt/vinzhub/.env with your credentials"
echo "2. Start the service: systemctl start vinzhub-api"
echo "3. Check status: systemctl status vinzhub-api"
echo "4. View logs: journalctl -u vinzhub-api -f"
echo ""
