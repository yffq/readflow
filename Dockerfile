FROM golang:1.24-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /readflow ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /readflow .
COPY template/ ./template/
COPY static/ ./static/
COPY migrations/ ./migrations/

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./readflow"]
