# syntax=docker/dockerfile:1.7

# ---------- Stage 1: frontend build ----------
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
# Pin pnpm to match packageManager in package.json. `corepack prepare --activate`
# pre-downloads the shim so `pnpm install` doesn't hit corepack's interactive
# signature prompt on first invocation.
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable && corepack prepare pnpm@9.15.0 --activate
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# ---------- Stage 2: backend build ----------
FROM golang:1.25-alpine AS backend
RUN apk add --no-cache build-base git
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist

ENV CGO_ENABLED=1
RUN go build -trimpath -ldflags="-s -w" -o /out/sub2balance main.go

# ---------- Stage 3: runtime ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -u 1000 app
WORKDIR /app
COPY --from=backend /out/sub2balance /app/sub2balance
RUN mkdir -p /app/data && chown -R app:app /app
USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --retries=3 --start-period=5s \
  CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["/app/sub2balance"]
