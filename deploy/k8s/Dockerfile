# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOARCH=amd64 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go mod tidy
RUN go build --ldflags "-extldflags -static" -o main ./cmd/server/main.go

# Runtime Stage
FROM alpine:latest

WORKDIR /app

ENV TZ=Asia/Manila

RUN apk add --no-cache tini tzdata

COPY --from=builder /app/main .

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/app/main"]