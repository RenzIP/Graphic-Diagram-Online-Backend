# --- Build stage ---
FROM golang:1.25 AS build
WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Build the WebSocket-only binary. cmd/ws wires just the collaboration hub
# (+ DB for membership auth and a /healthz probe) — no REST/OAuth/admin routes.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo -ldflags "-s -w" -o /out/gradiol-ws ./cmd/ws

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /out/gradiol-ws /app/gradiol-ws

# Cloud Run injects PORT; config.Load() already reads it (defaults to 8080).
EXPOSE 8080
ENTRYPOINT ["/app/gradiol-ws"]
