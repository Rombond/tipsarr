# trailer-resolver: yt-dlp + ffmpeg + Deno behind a tiny HTTP API. Optional companion of Tipsarr.
# Built locally by the owner; it is not published by the Tipsarr release workflow (see README.md for why).

# 1) the Go service (pure Go, built natively for the target architecture)
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS service
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /trailer-resolver .

# 2) the tools, downloaded from their official releases and checked against the published checksums
FROM debian:bookworm-slim AS tools
ARG TARGETARCH
ARG YTDLP_VERSION=latest
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl unzip xz-utils && rm -rf /var/lib/apt/lists/*
WORKDIR /dl
RUN set -eux; \
    case "$TARGETARCH" in \
      amd64) YT=yt-dlp_linux; FF=linux64; DENO=x86_64 ;; \
      arm64) YT=yt-dlp_linux_aarch64; FF=linuxarm64; DENO=aarch64 ;; \
      *) echo "unsupported architecture $TARGETARCH"; exit 1 ;; \
    esac; \
    if [ "$YTDLP_VERSION" = latest ]; then YTURL=https://github.com/yt-dlp/yt-dlp/releases/latest/download; else YTURL=https://github.com/yt-dlp/yt-dlp/releases/download/$YTDLP_VERSION; fi; \
    # yt-dlp: the SHA2-256SUMS of the same release
    curl -fsSL -o "$YT" "$YTURL/$YT"; curl -fsSL -o SHA2-256SUMS "$YTURL/SHA2-256SUMS"; \
    grep " $YT\$" SHA2-256SUMS | sha256sum -c -; \
    install -m 0755 "$YT" /yt-dlp; \
    # ffmpeg: static GPL build published by the yt-dlp project
    FFURL=https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest; \
    curl -fsSL -o ffmpeg.tar.xz "$FFURL/ffmpeg-master-latest-$FF-gpl.tar.xz"; curl -fsSL -o checksums.sha256 "$FFURL/checksums.sha256"; \
    echo "$(grep " ffmpeg-master-latest-$FF-gpl.tar.xz\$" checksums.sha256 | cut -d' ' -f1)  ffmpeg.tar.xz" | sha256sum -c -; \
    mkdir ff; tar -xJf ffmpeg.tar.xz -C ff --strip-components=2 --wildcards '*/bin/ffmpeg' '*/bin/ffprobe'; \
    install -m 0755 ff/ffmpeg /ffmpeg; install -m 0755 ff/ffprobe /ffprobe; \
    # Deno: official release with its .sha256sum
    DURL=https://github.com/denoland/deno/releases/latest/download; \
    curl -fsSL -o deno.zip "$DURL/deno-$DENO-unknown-linux-gnu.zip"; curl -fsSL -o deno.zip.sha256sum "$DURL/deno-$DENO-unknown-linux-gnu.zip.sha256sum"; \
    echo "$(cut -d' ' -f1 deno.zip.sha256sum)  deno.zip" | sha256sum -c -; \
    unzip -q deno.zip deno; install -m 0755 deno /deno

# 3) runtime: Debian slim (glibc, which the yt-dlp and Deno binaries need), no shell tools beyond the base, non-root
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 65532 --no-create-home --home-dir /tmp/home resolver \
    && mkdir -p /opt/resolver/bin /cache /tmp/home && chown -R 65532:65532 /opt/resolver /cache /tmp/home
# the tools sit in a folder owned by the service user, so "yt-dlp -U" can replace itself
COPY --from=tools --chown=65532:65532 /yt-dlp /ffmpeg /ffprobe /deno /opt/resolver/bin/
COPY --from=service /trailer-resolver /usr/local/bin/trailer-resolver
ENV PATH=/opt/resolver/bin:$PATH \
    HOME=/tmp/home XDG_CACHE_HOME=/tmp/home/.cache \
    RESOLVER_CACHE_DIR=/cache RESOLVER_ADDR=:8090 \
    RESOLVER_YTDLP=/opt/resolver/bin/yt-dlp RESOLVER_FFMPEG=/opt/resolver/bin/ffmpeg \
    RESOLVER_FFPROBE=/opt/resolver/bin/ffprobe RESOLVER_DENO=/opt/resolver/bin/deno
USER 65532:65532
VOLUME /cache
EXPOSE 8090
HEALTHCHECK --interval=60s --timeout=5s --start-period=10s CMD ["/usr/local/bin/trailer-resolver", "-healthcheck"]
ENTRYPOINT ["/usr/local/bin/trailer-resolver"]
