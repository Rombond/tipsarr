# syntax=docker/dockerfile:1
#
# Production build for TipsArr — a client-only SvelteKit SPA (adapter-static).
# Build stage compiles the static bundle; runtime stage serves it with nginx,
# which also reverse-proxies /seerr-api/* to the real Seerr instance (Seerr
# doesn't send CORS headers, so calls must go through a same-origin proxy —
# see deployments/nginx.conf.template and vite.config.ts's dev-server equivalent).
#
# Build from the repo root:
#   docker build -f deployments/Dockerfile -t tipsarr .
# (docker-compose.yml in this folder does this for you with the right build args.)

FROM oven/bun:1 AS build
WORKDIR /app

COPY package.json bun.lock ./
RUN bun install

COPY . .

# Baked into the static bundle at build time — the app calls these URLs
# directly from the browser. Changing any of these requires a rebuild.
# VITE_SEERR_URL is intentionally NOT needed here: the frontend only ever
# calls the same-origin /seerr-api path; the real Seerr URL is nginx's proxy
# target, configured at container *runtime* via SEERR_UPSTREAM instead.
ARG VITE_BOXARR_URL
ARG VITE_SUGGESTARR_URL
ARG VITE_RADARR_URL
ARG VITE_RADARR_API_KEY
ARG VITE_TMDB_API_URL
ARG VITE_TMDB_IMAGE_BASE
ARG VITE_TMDB_API_KEY
ENV VITE_BOXARR_URL=$VITE_BOXARR_URL \
    VITE_SUGGESTARR_URL=$VITE_SUGGESTARR_URL \
    VITE_RADARR_URL=$VITE_RADARR_URL \
    VITE_RADARR_API_KEY=$VITE_RADARR_API_KEY \
    VITE_TMDB_API_URL=$VITE_TMDB_API_URL \
    VITE_TMDB_IMAGE_BASE=$VITE_TMDB_IMAGE_BASE \
    VITE_TMDB_API_KEY=$VITE_TMDB_API_KEY

RUN bun run build

FROM nginx:1.27-alpine AS runtime
COPY --from=build /app/build /usr/share/nginx/html
COPY deployments/nginx.conf.template /etc/nginx/templates/default.conf.template

EXPOSE 80
