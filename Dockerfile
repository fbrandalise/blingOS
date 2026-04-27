# syntax=docker/dockerfile:1

# ── Stage 1: build web frontend ──────────────────────────────────────────────
FROM node:22-alpine AS webbuilder
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --ignore-scripts
COPY web/ ./
RUN npm run build

# ── Stage 2: build Go binary (with embedded web assets) ──────────────────────
FROM golang:1.26-alpine AS gobuilder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Inject the built web dist so it gets embedded via //go:embed all:web/dist
COPY --from=webbuilder /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /wuphf ./cmd/wuphf

# ── Stage 3: minimal runtime image ───────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache git ca-certificates tzdata && \
    git config --system user.email "office@wuphf.app" && \
    git config --system user.name "Wuphf Office"

COPY --from=gobuilder /wuphf /usr/local/bin/wuphf

# Wiki data lives here; mount a Railway Volume at this path for persistence.
VOLUME /root/.wuphf

ENV WUPHF_WEB_HOST=0.0.0.0
# Railway overrides PORT at runtime. Default matches --web-port default.
ENV PORT=7891

EXPOSE 7891

CMD ["wuphf", "--no-open", "--no-nex"]
