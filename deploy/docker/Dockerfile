# Stage 1: Development image (for Air)
FROM golang:1.24-alpine AS development

RUN apk add --no-cache git curl

# Install Air
RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b /usr/local/bin

WORKDIR /app

# Volume akan override ini, copy hanya biar build berhasil
COPY . .

RUN mkdir -p bin

CMD ["air", "-c", ".air.toml"]


# Stage 2: Production Builder
FROM golang:1.24-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOARCH=amd64 \
    GOOS=linux

WORKDIR /build

COPY go.mod go.sum ./
COPY .env ./
RUN go mod download
COPY . .

RUN mkdir -p .dist
RUN go build -o .dist/grls ./cmd/server/main.go
RUN go build --ldflags "-extldflags -static" -o .dist/grls ./cmd/server/main.go


# Stage 3: Production Runtime
FROM alpine:latest AS production

RUN apk add --no-cache tini curl

WORKDIR /app

COPY --from=builder /build/.env .env
COPY --from=builder /build/.dist /app/.dist

EXPOSE 50051

# Add healthcheck endpoint
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:50051/api/healthz || exit 1

# Use tini as init process
ENTRYPOINT ["/sbin/tini", "--"]

CMD ["./.dist/grls"]
