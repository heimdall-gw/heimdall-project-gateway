FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /heimdall-app .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /heimdall-app .
COPY config.yaml .

RUN chmod +x ./heimdall-app

EXPOSE 8080

CMD ["./heimdall-app"]
