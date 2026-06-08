# Lelemon MCP server — production image.
# The package is self-contained (deps are npm, no workspace links), so it builds
# standalone without the monorepo. `mcify build --bundle-deps` inlines all deps
# into a single dist/server.mjs, so the runtime stage needs only Node + that file.

FROM node:22-alpine AS build
WORKDIR /app
COPY package.json ./
RUN npm install --no-audit --no-fund
COPY . .
RUN npx mcify build --bundle-deps

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
# LELEMON_ENDPOINT defaults to https://api.lelemon.dev (see src/client.ts).
# PORT is provided by the host (Railway injects it); the server reads it.
COPY --from=build /app/dist/server.mjs ./server.mjs
EXPOSE 8888
CMD ["node", "server.mjs"]
