# VinzHub REST API v2

High-performance REST API for VinzHub inventory management.

## Features

- 🚀 Go 1.21+ with Chi router
- 🗄️ PostgreSQL (Aiven) or SQLite for inventory storage
- ⚡ Redis write-behind buffer for high throughput
- 🔐 Token-based authentication
- 📊 Admin dashboard

## Quick Deploy (VPS)

```bash
# Clone repo
git clone https://github.com/Sezy0/apis-vhz-v2.git
cd apis-vhz-v2

# Run install script
chmod +x deploy/install.sh
sudo ./deploy/install.sh

# Edit config
sudo nano /opt/vinzhub/.env

# Start service
sudo systemctl start vinzhub-api
sudo systemctl status vinzhub-api
```

## Configuration

Edit `/opt/vinzhub/.env`:

```env
# Server
SERVER_PORT=8080

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# PostgreSQL
INVENTORY_DB_TYPE=postgres
INVENTORY_DB_HOST=your-host
INVENTORY_DB_PORT=5432
INVENTORY_DB_NAME=your-db
INVENTORY_DB_USER=your-user
INVENTORY_DB_PASS=your-pass
INVENTORY_DB_SSLMODE=require

# MySQL (key_accounts)
DB_HOST=your-mysql-host
DB_PORT=3306
DB_NAME=your-db
DB_USER=your-user
DB_PASS=your-pass

# API Keys
API_KEYS=your-api-key
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check |
| POST | `/api/v1/auth/token` | Generate token |
| POST | `/api/v1/inventory/{id}/sync` | Sync inventory |
| GET | `/api/v1/inventory/{id}` | Get inventory |
| GET | `/api/v1/admin/stats` | Admin stats |
| GET | `/admin` | Admin dashboard |

## Logs

```bash
# View logs
sudo journalctl -u vinzhub-api -f

# Restart
sudo systemctl restart vinzhub-api
```
