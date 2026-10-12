# Stage 1: Build the Go binary
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./main.go

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/server /app/server
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080

CMD ["/app/server"]
