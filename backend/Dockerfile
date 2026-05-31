# syntax=docker/dockerfile:1.7

FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api

# debian-slim (not distroless/static) because the API converts uploaded voice
# messages with ffmpeg (m4a/aac -> OggOpus for SpeechKit); ffmpeg needs a base
# with a package manager. The Go binary is still static (CGO disabled).
FROM debian:12-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system app && useradd --system --gid app --no-create-home app
COPY --from=builder /out/api /usr/local/bin/api
USER app:app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
