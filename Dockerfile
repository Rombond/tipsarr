# 1) build the Svelte SPA
# the SPA is the same on every architecture, so it is always built natively (never under emulation)
# node:25-alpine (digest updated by Dependabot)
FROM --platform=$BUILDPLATFORM node:25-alpine@sha256:bdf2cca6fe3dabd014ea60163eca3f0f7015fbd5c7ee1b0e9ccb4ced6eb02ef4 AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# the web app reads the translation files from ../locales
COPY locales/ /app/locales/
RUN npm run build

# 2) build the Go binary with the SPA embedded (pure Go, no CGO)
# cross-compiled natively for the target architecture (no QEMU)
# golang:1.26-alpine
FROM --platform=$BUILDPLATFORM golang:1.26-alpine@sha256:87c65a688975c0b1d4585d204874c45ca78354670e3a5f11368c5a492303d696 AS backend
ARG TARGETOS
ARG TARGETARCH
# the release tag, shown in the apps (Settings > server version)
ARG VERSION=dev
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /app/frontend/build ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X github.com/Rombond/tipsarr/backend/internal/api.Version=${VERSION}" -o /tipsarr ./cmd/tipsarr
RUN mkdir /config-empty

# 3) tiny runtime
# gcr.io/distroless/static-debian12:latest
FROM gcr.io/distroless/static-debian12:latest@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2
COPY --from=backend /tipsarr /tipsarr
# Starts as root only to give /config to PUID:PGID (default 65532), then runs as that user (see
# backend/cmd/tipsarr/privs_linux.go). Needs only CHOWN, SETUID, SETGID: see docs/installation.md "Hardening".
COPY --from=backend --chown=65532:65532 /config-empty /config
ENV TIPSARR_CONFIG_DIR=/config TIPSARR_PORT=8080
VOLUME /config
EXPOSE 8080
ENTRYPOINT ["/tipsarr"]
