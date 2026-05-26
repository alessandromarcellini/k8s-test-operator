# Build stage
FROM golang:1.26.3-alpine3.23 AS builder

WORKDIR /app

# copy only dependencies first to improve layer caching
COPY go.mod go.sum ./

# install dependecies
RUN go mod download

# copy code
COPY . .

COPY ./cert ./cert

RUN go build -o proxy .


# ------------
# ------------
# ------------


# Runtime stage
FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/proxy .
COPY --from=builder /app/cert ./cert

EXPOSE 8080

CMD ["./proxy"]