# syntax=docker/dockerfile:1

# ---- build stage -------------------------------------------------------
FROM golang:1.23-alpine AS build

WORKDIR /src

# Cache module downloads first for faster incremental builds.
COPY go.mod ./
RUN go mod download

COPY . .

# Static, CGO-free Linux binary.
ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath -ldflags="-s -w" \
    -o /out/marine-farm-environment-service .

# ---- runtime stage -----------------------------------------------------
FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=build /out/marine-farm-environment-service /app/marine-farm-environment-service

# Runtime defaults can be overridden with env vars.
ENV PORT=8080
ENV DATA_FILE=/app/data/marine_data.json
RUN mkdir -p /app/data && chown -R app:app /app/data

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:${PORT}/healthz || exit 1

ENTRYPOINT ["/app/marine-farm-environment-service"]
