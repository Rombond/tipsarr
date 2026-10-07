# 1) build the Svelte SPA
# the SPA is the same on every architecture, so it is always built natively (never under emulation)
FROM --platform=$BUILDPLATFORM node:25-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 2) build the Go binary with the SPA embedded (pure Go, no CGO)
# cross-compiled natively for the target architecture (no QEMU)
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS backend
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /app/frontend/build ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /tipsarr ./cmd/tipsarr
RUN mkdir /config-empty

# 3) tiny runtime
FROM gcr.io/distroless/static-debian12
COPY --from=backend /tipsarr /tipsarr
# Starts as root only to give /config to PUID:PGID (default 65532), then runs as that user
COPY --from=backend --chown=65532:65532 /config-empty /config
ENV TIPSARR_CONFIG_DIR=/config TIPSARR_PORT=8080
VOLUME /config
EXPOSE 8080
ENTRYPOINT ["/tipsarr"]
