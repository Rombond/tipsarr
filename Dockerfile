# 1) build the Svelte SPA
FROM node:25-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 2) build the Go binary with the SPA embedded (pure Go, no CGO)
FROM golang:1.26-alpine AS backend
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /app/frontend/build ./internal/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tipsarr ./cmd/tipsarr
RUN mkdir /config-empty

# 3) tiny runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /tipsarr /tipsarr
# the distroless user (65532) must own /config so the SQLite file and caches can be created
COPY --from=backend --chown=65532:65532 /config-empty /config
ENV TIPSARR_CONFIG_DIR=/config TIPSARR_PORT=8080
VOLUME /config
EXPOSE 8080
ENTRYPOINT ["/tipsarr"]
