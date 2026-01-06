# Deploy en Railway

## Estructura

El proyecto tiene 2 servicios:
- `apps/server` - API Go (puerto 8080)
- `apps/web` - Dashboard Next.js (puerto 3000)

Cada servicio tiene su `railway.toml` configurado.

## Opción 1: Desde la UI de Railway (Recomendado)

### 1. Crear proyecto

1. Ir a [railway.app](https://railway.app) → New Project
2. Seleccionar "Deploy from GitHub repo"
3. Conectar tu repositorio

### 2. Agregar PostgreSQL

1. En el proyecto, click "New" → "Database" → "PostgreSQL"
2. Railway crea `DATABASE_URL` automáticamente

### 3. Crear servicio API

1. Click "New" → "GitHub Repo" → Seleccionar el mismo repo
2. En Settings del servicio:
   - **Root Directory**: `apps/server`
   - **Watch Paths**: `apps/server/**`
3. Variables de entorno:
   ```
   PORT=8080
   JWT_SECRET=<genera-un-secret-seguro-de-32+-chars>
   FRONTEND_URL=https://<tu-web>.up.railway.app
   LOG_LEVEL=info
   ```
4. En Networking → Generate Domain

### 4. Crear servicio Web

1. Click "New" → "GitHub Repo" → Seleccionar el mismo repo
2. En Settings del servicio:
   - **Root Directory**: `apps/web`
   - **Watch Paths**: `apps/web/**`
3. Variables de entorno:
   ```
   NEXT_PUBLIC_API_URL=https://<tu-api>.up.railway.app
   ```
4. En Networking → Generate Domain

## Opción 2: Desde CLI

```bash
# Instalar CLI
npm install -g @railway/cli

# Login
railway login

# Crear proyecto
railway init
```

### Desplegar API

```bash
cd apps/server
railway link  # Seleccionar o crear servicio
railway variables set PORT=8080
railway variables set JWT_SECRET=tu-secret-seguro
railway variables set FRONTEND_URL=https://tu-web.up.railway.app
railway up
```

### Desplegar Web

```bash
cd apps/web
railway link
railway variables set NEXT_PUBLIC_API_URL=https://tu-api.up.railway.app
railway up
```

## Variables de entorno

### API (apps/server)

| Variable | Requerida | Descripción |
|----------|-----------|-------------|
| `DATABASE_URL` | ✅ | Auto-provista por Railway PostgreSQL |
| `PORT` | ✅ | `8080` |
| `JWT_SECRET` | ✅ | Secret para JWT (mínimo 32 chars) |
| `FRONTEND_URL` | ✅ | URL del dashboard para CORS |
| `LOG_LEVEL` | ❌ | `debug`, `info`, `warn`, `error` |
| `GOOGLE_CLIENT_ID` | ❌ | Para OAuth con Google |
| `GOOGLE_CLIENT_SECRET` | ❌ | Para OAuth con Google |

### Web (apps/web)

| Variable | Requerida | Descripción |
|----------|-----------|-------------|
| `NEXT_PUBLIC_API_URL` | ✅ | URL pública de la API |

## Configuración (railway.toml)

Cada servicio tiene su configuración:

**apps/server/railway.toml**
```toml
[build]
builder = "DOCKERFILE"
dockerfilePath = "Dockerfile"
watchPatterns = ["apps/server/**"]

[deploy]
healthcheckPath = "/health"
healthcheckTimeout = 30
restartPolicyType = "ON_FAILURE"
restartPolicyMaxRetries = 3
```

**apps/web/railway.toml**
```toml
[build]
builder = "DOCKERFILE"
dockerfilePath = "Dockerfile"
watchPatterns = ["apps/web/**"]

[deploy]
healthcheckPath = "/"
healthcheckTimeout = 30
restartPolicyType = "ON_FAILURE"
restartPolicyMaxRetries = 3
```

## Notas importantes

- **Watch Patterns**: Evita rebuilds innecesarios cuando cambias código en otro servicio
- **PostgreSQL**: El servidor auto-detecta PostgreSQL vs SQLite por el prefijo de `DATABASE_URL`
- **Health checks**: La API expone `/health`, el web usa `/`

## Costos

- **Hobby**: $5/mes - 512MB RAM, shared CPU, 1GB PostgreSQL
- **Pro**: $20/mes + uso - Recursos escalables

## Links

- [Railway Monorepo Guide](https://docs.railway.com/guides/monorepo)
- [Railway Dockerfile Guide](https://docs.railway.com/guides/dockerfiles)
- [Railway Config Reference](https://docs.railway.com/reference/config-as-code)
