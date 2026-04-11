# --- Build stage ---
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /laightdb ./cmd/laightdb

# --- Production stage ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /laightdb /usr/local/bin/laightdb
EXPOSE 8080
VOLUME ["/data"]
ENV LAIGHTDB_DATA_DIR=/data
ENTRYPOINT ["/usr/local/bin/laightdb"]
