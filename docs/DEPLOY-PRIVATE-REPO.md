# Lelemon Deploy (Repo Privado)

Este documento describe cómo configurar el repositorio privado `lelemon-deploy` para gestionar los deployments de producción.

## Estructura del Repo Privado

```
lelemon-deploy/
├── .github/
│   └── workflows/
│       ├── deploy-api.yml        # Deploy API a Droplet
│       └── deploy-web.yml        # Deploy Web a Railway (opcional, Railway ya auto-deploya)
│
├── docker/
│   ├── Dockerfile.api            # Dockerfile EE (copiado de ee/server/Dockerfile)
│   └── docker-compose.prod.yml   # Stack completo: API + PG + ClickHouse
│
├── scripts/
│   ├── deploy.sh                 # Script principal de deploy
│   ├── backup-db.sh              # Backup PostgreSQL + ClickHouse
│   └── rollback.sh               # Rollback a versión anterior
│
├── nginx/
│   └── api.conf                  # Config nginx para reverse proxy
│
├── .env.example                  # Template de variables
└── README.md
```

## Archivos Clave

### `.github/workflows/deploy-api.yml`

```yaml
name: Deploy API

on:
  push:
    branches: [main]
  workflow_dispatch:
    inputs:
      version:
        description: 'Version to deploy (default: latest)'
        required: false
        default: 'latest'

env:
  DROPLET_HOST: ${{ secrets.DROPLET_HOST }}
  DROPLET_USER: ${{ secrets.DROPLET_USER }}

jobs:
  deploy:
    name: Deploy to Droplet
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Setup SSH
        uses: webfactory/ssh-agent@v0.9.0
        with:
          ssh-private-key: ${{ secrets.DROPLET_SSH_KEY }}

      - name: Add host to known_hosts
        run: ssh-keyscan -H ${{ env.DROPLET_HOST }} >> ~/.ssh/known_hosts

      - name: Clone/Update lelemon repo on server
        run: |
          ssh ${{ env.DROPLET_USER }}@${{ env.DROPLET_HOST }} << 'EOF'
            cd /opt/lelemon

            # Clone or update main repo
            if [ ! -d "lelemon" ]; then
              git clone https://github.com/lelemondev/lelemon.git
            fi
            cd lelemon
            git fetch origin
            git checkout main
            git pull origin main
          EOF

      - name: Copy private configs
        run: |
          scp docker/Dockerfile.api ${{ env.DROPLET_USER }}@${{ env.DROPLET_HOST }}:/opt/lelemon/
          scp docker/docker-compose.prod.yml ${{ env.DROPLET_USER }}@${{ env.DROPLET_HOST }}:/opt/lelemon/

      - name: Build and deploy
        run: |
          ssh ${{ env.DROPLET_USER }}@${{ env.DROPLET_HOST }} << 'EOF'
            cd /opt/lelemon

            # Build EE image
            docker build -f Dockerfile.api -t lelemon-api:latest ./lelemon

            # Deploy with zero downtime
            docker-compose -f docker-compose.prod.yml up -d --no-deps --build api

            # Cleanup old images
            docker image prune -f
          EOF

      - name: Health check
        run: |
          sleep 10
          curl -f https://api.lelemon.dev/health || exit 1

      - name: Notify on failure
        if: failure()
        run: |
          # Slack/Discord webhook notification
          echo "Deploy failed!"
```

### `docker/Dockerfile.api`

```dockerfile
# Lelemon Enterprise Server
# Este archivo NO está en el repo público

FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go workspace
COPY lelemon/go.work ./
COPY lelemon/apps/server ./apps/server
COPY lelemon/ee/server ./ee/server

# Build
WORKDIR /app/ee/server
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
RUN mkdir -p /data && chown appuser:appuser /data

WORKDIR /app
COPY --from=builder /server .
USER appuser

EXPOSE 8080
ENV PORT=8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./server"]
```

### `docker/docker-compose.prod.yml`

```yaml
version: '3.8'

services:
  api:
    image: lelemon-api:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - DATABASE_URL=postgres://lelemon:${PG_PASSWORD}@postgres:5432/lelemon?sslmode=disable
      - ANALYTICS_DATABASE_URL=clickhouse://default:${CH_PASSWORD}@clickhouse:9000/lelemon
      - JWT_SECRET=${JWT_SECRET}
      - JWT_EXPIRATION=24h
      - FRONTEND_URL=https://app.lelemon.dev
      - GOOGLE_CLIENT_ID=${GOOGLE_CLIENT_ID}
      - GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET}
      - GOOGLE_REDIRECT_URL=https://api.lelemon.dev/api/v1/auth/google/callback
      # Enterprise
      - LEMONSQUEEZY_API_KEY=${LEMONSQUEEZY_API_KEY}
      - LEMONSQUEEZY_WEBHOOK_SECRET=${LEMONSQUEEZY_WEBHOOK_SECRET}
      - LEMONSQUEEZY_STORE_ID=${LEMONSQUEEZY_STORE_ID}
      - LEMONSQUEEZY_PRO_VARIANT_ID=${LEMONSQUEEZY_PRO_VARIANT_ID}
      - LEMONSQUEEZY_ENTERPRISE_VARIANT_ID=${LEMONSQUEEZY_ENTERPRISE_VARIANT_ID}
    depends_on:
      postgres:
        condition: service_healthy
      clickhouse:
        condition: service_healthy
    networks:
      - lelemon

  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      - POSTGRES_USER=lelemon
      - POSTGRES_PASSWORD=${PG_PASSWORD}
      - POSTGRES_DB=lelemon
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U lelemon"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - lelemon

  clickhouse:
    image: clickhouse/clickhouse-server:24.3
    restart: unless-stopped
    environment:
      - CLICKHOUSE_USER=default
      - CLICKHOUSE_PASSWORD=${CH_PASSWORD}
      - CLICKHOUSE_DB=lelemon
    volumes:
      - ch_data:/var/lib/clickhouse
    healthcheck:
      test: ["CMD", "clickhouse-client", "--query", "SELECT 1"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - lelemon

volumes:
  pg_data:
  ch_data:

networks:
  lelemon:
    driver: bridge
```

### `.env.example`

```bash
# PostgreSQL
PG_PASSWORD=your-secure-password

# ClickHouse
CH_PASSWORD=your-secure-password

# JWT
JWT_SECRET=your-very-long-random-secret-at-least-32-chars

# Google OAuth
GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=xxx

# Lemon Squeezy (Enterprise billing)
LEMONSQUEEZY_API_KEY=xxx
LEMONSQUEEZY_WEBHOOK_SECRET=xxx
LEMONSQUEEZY_STORE_ID=xxx
LEMONSQUEEZY_PRO_VARIANT_ID=xxx
LEMONSQUEEZY_ENTERPRISE_VARIANT_ID=xxx
```

### `scripts/deploy.sh`

```bash
#!/bin/bash
set -e

echo "🍋 Lelemon Deploy Script"
echo "========================"

# Load environment
source /opt/lelemon/.env

# Pull latest code
cd /opt/lelemon/lelemon
git pull origin main

# Build API
echo "Building API..."
docker build -f ../Dockerfile.api -t lelemon-api:latest .

# Deploy with zero downtime
echo "Deploying..."
cd /opt/lelemon
docker-compose -f docker-compose.prod.yml up -d --no-deps api

# Wait for health
echo "Waiting for health check..."
sleep 10
curl -f http://localhost:8080/health || exit 1

echo "✅ Deploy complete!"
```

### `scripts/backup-db.sh`

```bash
#!/bin/bash
set -e

BACKUP_DIR="/opt/lelemon/backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p $BACKUP_DIR

# Backup PostgreSQL
echo "Backing up PostgreSQL..."
docker exec lelemon-postgres pg_dump -U lelemon lelemon > $BACKUP_DIR/postgres.sql

# Backup ClickHouse
echo "Backing up ClickHouse..."
docker exec lelemon-clickhouse clickhouse-client --query "SELECT * FROM lelemon.traces FORMAT Native" > $BACKUP_DIR/traces.native
docker exec lelemon-clickhouse clickhouse-client --query "SELECT * FROM lelemon.spans FORMAT Native" > $BACKUP_DIR/spans.native

# Compress
tar -czf $BACKUP_DIR.tar.gz -C $BACKUP_DIR .
rm -rf $BACKUP_DIR

echo "✅ Backup saved to $BACKUP_DIR.tar.gz"
```

## GitHub Secrets Necesarios

En el repo `lelemon-deploy`, configura estos secrets:

| Secret | Descripción |
|--------|-------------|
| `DROPLET_HOST` | IP o dominio del droplet |
| `DROPLET_USER` | Usuario SSH (ej: `deploy`) |
| `DROPLET_SSH_KEY` | Private key para SSH |

## Flujo de Deploy

```
1. Push a main en lelemon-deploy
       │
       ▼
2. GitHub Action se ejecuta
       │
       ▼
3. SSH al Droplet
       │
       ▼
4. Pull código de lelemon (público)
       │
       ▼
5. Build imagen EE con Dockerfile privado
       │
       ▼
6. docker-compose up (zero downtime)
       │
       ▼
7. Health check
       │
       ▼
8. ✅ Done
```

## Railway (Frontend)

Railway ya auto-deploya desde el repo público cuando hay push a `main`.
No necesitas workflow adicional a menos que quieras control manual.

Si quieres deploy manual:

```yaml
# .github/workflows/deploy-web.yml (opcional)
name: Deploy Web

on:
  workflow_dispatch:

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Trigger Railway deploy
        run: |
          curl -X POST "${{ secrets.RAILWAY_WEBHOOK_URL }}"
```

## Comandos Útiles

```bash
# En el Droplet

# Ver logs
docker-compose -f docker-compose.prod.yml logs -f api

# Restart
docker-compose -f docker-compose.prod.yml restart api

# Ver estado
docker-compose -f docker-compose.prod.yml ps

# Ejecutar migrations manualmente
docker exec lelemon-api ./server migrate

# Backup manual
./scripts/backup-db.sh
```
