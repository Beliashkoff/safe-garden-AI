# syntax=docker/dockerfile:1.7

# Debian-based (glibc) builder so the produced binary matches the debian runtime
# below: whether the build ends up static (CGO off) or glibc-dynamic, it execs in
# debian:12-slim. An alpine/musl build does NOT run in debian (no musl loader) —
# that mismatch caused "exec /usr/local/bin/api: no such file or directory".
FROM golang:1.27-bookworm AS builder
ENV CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# debian-slim (not distroless/static) because the API converts uploaded voice
# messages with ffmpeg (m4a/aac -> OggOpus for SpeechKit); ffmpeg needs a base
# with a package manager.
FROM debian:12-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg ca-certificates wget \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system app && useradd --system --gid app --no-create-home app
COPY --from=builder /out/api /usr/local/bin/api
USER app:app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
