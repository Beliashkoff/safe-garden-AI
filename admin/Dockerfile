# syntax=docker/dockerfile:1.7

# Build the SPA in a node stage, serve the static bundle from caddy:2-alpine.
# Built in CI (GitHub runners reach the npm registry; the RU VM does not need
# node). The edge Caddy proxies admin.<domain> to this container.
FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM caddy:2-alpine
COPY --from=build /app/dist /srv
COPY Caddyfile /etc/caddy/Caddyfile
EXPOSE 80
