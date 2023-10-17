FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:3.9

RUN adduser -D -g '' appuser
USER appuser

WORKDIR /home/appuser
COPY --from=builder /app/main .
ENTRYPOINT ["./main"]
