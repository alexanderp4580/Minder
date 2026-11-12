FROM docker.io/library/debian:bookworm-slim AS flutter
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl git unzip xz-utils libglu1-mesa && rm -rf /var/lib/apt/lists/*
RUN git clone --depth 1 --branch 3.47.6 https://github.com/flutter/flutter.git /opt/flutter
ENV PATH="/opt/flutter/bin:${PATH}"
WORKDIR /build/app
COPY app/pubspec.yaml app/pubspec.lock ./
RUN flutter pub get
COPY app/ ./
RUN flutter build web --release --no-wasm-dry-run

FROM docker.io/library/golang:1.27.2-bookworm AS server
WORKDIR /build
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=1 go build -trimpath -ldflags='-s -w' -o /minder ./cmd/minder

FROM docker.io/library/debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/* && useradd -u 10001 -m minder && mkdir -p /data /srv && chown minder:minder /data /srv
COPY --from=server /minder /usr/local/bin/minder
COPY --from=flutter /build/app/build/web /srv/web
COPY brand/index.html brand/app-icon.svg brand/app-icon.png /srv/branding/
ENV ENV=production AUTH_MODE=cloudflare DATABASE_PATH=/data/minder.db WEB_DIR=/srv/web BRAND_DIR=/srv/branding LISTEN_ADDR=0.0.0.0:8080
USER minder
WORKDIR /srv
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s CMD curl --fail --silent http://127.0.0.1:8080/v1/health || exit 1
CMD ["minder"]
