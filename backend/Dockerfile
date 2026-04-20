# Stage 1 — build a static binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# TARGETOS / TARGETARCH are injected by `docker buildx build --platform`
# so the same Dockerfile produces amd64 (laptop) and arm64 (Raspberry Pi) images.
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /bin/api ./cmd/api

# Stage 2 — minimal runtime image (no Go toolchain, ~5 MB)
FROM alpine:3.19

RUN apk add --no-cache tzdata ca-certificates

COPY --from=builder /bin/api /api

EXPOSE 8080

ENTRYPOINT ["/api"]
