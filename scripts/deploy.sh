#!/bin/bash
# Lelemon Production Deployment Script
# Run this on your DigitalOcean droplet

set -e

echo "=== Lelemon Deployment Script ==="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (sudo ./deploy.sh)"
  exit 1
fi

# ===========================================
# 1. Install Docker if not present
# ===========================================
if ! command -v docker &> /dev/null; then
  echo "Installing Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl enable docker
  systemctl start docker
  echo "Docker installed successfully"
else
  echo "Docker already installed: $(docker --version)"
fi

# ===========================================
# 2. Install Docker Compose plugin if not present
# ===========================================
if ! docker compose version &> /dev/null; then
  echo "Installing Docker Compose plugin..."
  apt-get update
  apt-get install -y docker-compose-plugin
  echo "Docker Compose installed successfully"
else
  echo "Docker Compose already installed: $(docker compose version)"
fi

# ===========================================
# 3. Create app directory
# ===========================================
APP_DIR="/opt/lelemon"
mkdir -p $APP_DIR
cd $APP_DIR

# ===========================================
# 4. Create docker-compose.yml
# ===========================================
echo "Creating docker-compose.yml..."
cat > docker-compose.yml << 'COMPOSE_EOF'
services:
  lelemon-api:
    image: ghcr.io/lelemondev/lelemon/server:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - DATABASE_URL=postgres://lelemon:${POSTGRES_PASSWORD}@postgres:5432/lelemon?sslmode=disable
      - ENVIRONMENT=production
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - LOG_FORMAT=json
      - JWT_SECRET=${JWT_SECRET}
      - FRONTEND_URL=${FRONTEND_URL}
      - ALLOWED_ORIGINS=${ALLOWED_ORIGINS}
      - GOOGLE_CLIENT_ID=${GOOGLE_CLIENT_ID:-}
      - GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET:-}
      - GOOGLE_REDIRECT_URL=${GOOGLE_REDIRECT_URL:-}
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

  postgres:
    image: postgres:16-alpine
    environment:
      - POSTGRES_USER=lelemon
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=lelemon
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U lelemon -d lelemon"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped

volumes:
  postgres_data:
COMPOSE_EOF

# ===========================================
# 5. Create .env file if not exists
# ===========================================
if [ ! -f .env ]; then
  echo ""
  echo "Creating .env file..."
  echo "Please provide the following values:"
  echo ""

  # Generate secure defaults
  DEFAULT_PG_PASS=$(openssl rand -hex 16)
  DEFAULT_JWT_SECRET=$(openssl rand -hex 32)

  read -p "Frontend URL (e.g., https://lelemon.railway.app): " FRONTEND_URL
  read -p "PostgreSQL password [auto-generated]: " PG_PASS
  PG_PASS=${PG_PASS:-$DEFAULT_PG_PASS}

  read -p "JWT Secret [auto-generated]: " JWT_SEC
  JWT_SEC=${JWT_SEC:-$DEFAULT_JWT_SECRET}

  cat > .env << ENV_EOF
# Generated on $(date)
POSTGRES_PASSWORD=$PG_PASS
JWT_SECRET=$JWT_SEC
FRONTEND_URL=$FRONTEND_URL
ALLOWED_ORIGINS=$FRONTEND_URL
LOG_LEVEL=info

# Optional: Google OAuth
# GOOGLE_CLIENT_ID=
# GOOGLE_CLIENT_SECRET=
# GOOGLE_REDIRECT_URL=
ENV_EOF

  echo ""
  echo ".env file created with secure random values"
  echo "IMPORTANT: Save these values somewhere safe!"
  echo "  PostgreSQL Password: $PG_PASS"
  echo "  JWT Secret: $JWT_SEC"
else
  echo ".env file already exists, skipping..."
fi

# ===========================================
# 6. Pull and start services
# ===========================================
echo ""
echo "Pulling latest images..."
docker compose pull

echo ""
echo "Starting services..."
docker compose up -d

# ===========================================
# 7. Wait for health check
# ===========================================
echo ""
echo "Waiting for services to be healthy..."
sleep 10

if docker compose ps | grep -q "healthy"; then
  echo ""
  echo "=== Deployment Successful! ==="
  echo ""
  echo "API is running at: http://$(curl -s ifconfig.me):8080"
  echo ""
  echo "Health check: curl http://localhost:8080/health"
  echo ""
  echo "View logs: docker compose logs -f"
  echo "Stop: docker compose down"
  echo "Update: docker compose pull && docker compose up -d"
else
  echo ""
  echo "Warning: Services may not be fully healthy yet."
  echo "Check logs with: docker compose logs"
fi
